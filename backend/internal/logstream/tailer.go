package logstream

import (
	"bufio"
	"context"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Message 是 tailer 发给订阅者的消息。
type Message struct {
	Type   string `json:"type"`
	Entry  Entry  `json:"entry,omitempty"`
	Cursor int64  `json:"cursor,omitempty"`
	Reason string `json:"reason,omitempty"`
}

// Manager 管理按文件路径共享的 tailer。
type Manager struct {
	mu      sync.Mutex
	tailers map[string]*Tailer
}

// GlobalManager 是通用日志和同步任务详情共享的 tailer manager。
var GlobalManager = NewManager()

// NewManager 创建共享 tailer manager。
func NewManager() *Manager {
	return &Manager{tailers: make(map[string]*Tailer)}
}

// Subscribe 订阅指定日志文件的新行。
func (m *Manager) Subscribe(ctx context.Context, path string, startCursor int64, buffer int) (<-chan Message, func(), error) {
	canonical, err := filepath.Abs(path)
	if err != nil {
		return nil, nil, err
	}
	if buffer <= 0 {
		buffer = 128
	}

	m.mu.Lock()
	tailer := m.tailers[canonical]
	if tailer == nil {
		tailer = newTailer(canonical, startCursor, func() {
			m.mu.Lock()
			delete(m.tailers, canonical)
			m.mu.Unlock()
		})
		m.tailers[canonical] = tailer
		go tailer.run()
	}
	m.mu.Unlock()

	ch, unsubscribe := tailer.subscribe(ctx, startCursor, buffer)
	return ch, unsubscribe, nil
}

// TailerCount 返回当前 tailer 数量，用于测试和诊断。
func (m *Manager) TailerCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.tailers)
}

type subscriber struct {
	ch     chan Message
	cursor int64
}

// Tailer 负责单个文件路径的读取和广播。
type Tailer struct {
	path     string
	done     chan struct{}
	stopOnce sync.Once
	onStop   func()

	mu       sync.Mutex
	subs     map[*subscriber]struct{}
	cursor   int64
	leftover []byte
}

func newTailer(path string, startCursor int64, onStop func()) *Tailer {
	return &Tailer{
		path:   path,
		done:   make(chan struct{}),
		onStop: onStop,
		subs:   make(map[*subscriber]struct{}),
		cursor: startCursor,
	}
}

func (t *Tailer) subscribe(ctx context.Context, startCursor int64, buffer int) (<-chan Message, func()) {
	sub := &subscriber{ch: make(chan Message, buffer), cursor: startCursor}
	t.mu.Lock()
	t.subs[sub] = struct{}{}
	currentCursor := t.cursor
	t.mu.Unlock()

	if startCursor < currentCursor {
		go t.sendCatchUp(sub, startCursor)
	}

	var once sync.Once
	unsubscribe := func() {
		once.Do(func() {
			t.unsubscribe(sub)
		})
	}

	go func() {
		<-ctx.Done()
		unsubscribe()
	}()
	return sub.ch, unsubscribe
}

func (t *Tailer) sendCatchUp(sub *subscriber, startCursor int64) {
	entries, nextCursor, err := ReadEntriesFromCursor(t.path, startCursor, 500)
	if err != nil {
		select {
		case sub.ch <- Message{Type: "resync_required", Reason: err.Error()}:
		default:
		}
		return
	}
	for _, entry := range entries {
		select {
		case sub.ch <- Message{Type: "log_append", Entry: entry, Cursor: entry.Cursor}:
			sub.cursor = entry.Cursor
		default:
			t.unsubscribe(sub)
			return
		}
	}
	if nextCursor < t.cursor {
		select {
		case sub.ch <- Message{Type: "resync_required", Reason: "catch_up_limit_reached"}:
		default:
		}
	}
}

func (t *Tailer) unsubscribe(sub *subscriber) {
	t.mu.Lock()
	if _, ok := t.subs[sub]; ok {
		delete(t.subs, sub)
		close(sub.ch)
	}
	empty := len(t.subs) == 0
	t.mu.Unlock()
	if empty {
		t.stop()
	}
}

func (t *Tailer) stop() {
	t.stopOnce.Do(func() {
		close(t.done)
		if t.onStop != nil {
			t.onStop()
		}
	})
}

func (t *Tailer) run() {
	ticker := time.NewTicker(250 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-t.done:
			return
		case <-ticker.C:
			t.readAvailable()
		}
	}
}

func (t *Tailer) readAvailable() {
	file, err := os.Open(t.path)
	if err != nil {
		if os.IsNotExist(err) {
			t.broadcast(Message{Type: "resync_required", Reason: "log_file_missing"})
			return
		}
		t.broadcast(Message{Type: "error", Reason: err.Error()})
		return
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		t.broadcast(Message{Type: "error", Reason: err.Error()})
		return
	}
	if stat.Size() < t.cursor {
		t.cursor = 0
		t.leftover = nil
		t.broadcast(Message{Type: "resync_required", Reason: "log_file_truncated"})
	}
	if _, err := file.Seek(t.cursor, io.SeekStart); err != nil {
		t.broadcast(Message{Type: "error", Reason: err.Error()})
		return
	}

	reader := bufio.NewReader(file)
	for {
		line, err := reader.ReadBytes('\n')
		if len(line) > 0 {
			t.cursor += int64(len(line))
			if line[len(line)-1] == '\n' {
				content := append(t.leftover, line[:len(line)-1]...)
				t.leftover = nil
				entry := ParseLine(string(content))
				entry.Cursor = t.cursor
				t.broadcast(Message{Type: "log_append", Entry: entry, Cursor: t.cursor})
			} else {
				t.leftover = append(t.leftover, line...)
			}
		}
		if err == io.EOF {
			return
		}
		if err != nil {
			t.broadcast(Message{Type: "error", Reason: err.Error()})
			return
		}
	}
}

func (t *Tailer) broadcast(msg Message) {
	t.mu.Lock()
	defer t.mu.Unlock()
	for sub := range t.subs {
		select {
		case sub.ch <- msg:
			if msg.Cursor > 0 {
				sub.cursor = msg.Cursor
			}
		default:
			select {
			case sub.ch <- Message{Type: "resync_required", Reason: "subscriber_queue_full"}:
			default:
			}
			delete(t.subs, sub)
			close(sub.ch)
		}
	}
}
