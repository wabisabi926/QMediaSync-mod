package websocket

import (
	"sync"
	"time"
)

// SyncTaskEventPayload 是同步任务结构化事件数据。
type SyncTaskEventPayload struct {
	SyncID       uint   `json:"sync_id"`
	SyncPathID   uint   `json:"sync_path_id"`
	Status       int    `json:"status"`
	SubStatus    int    `json:"sub_status"`
	Total        int    `json:"total"`
	NewStrm      int    `json:"new_strm"`
	NewMeta      int    `json:"new_meta"`
	NewUpload    int    `json:"new_upload"`
	FinishAt     int64  `json:"finish_at"`
	LogPath      string `json:"log_path"`
	Sequence     uint64 `json:"sequence"`
	EventTime    int64  `json:"event_time"`
	CreatedAt    int64  `json:"created_at,omitempty"`
	UpdatedAt    int64  `json:"updated_at,omitempty"`
	LocalPath    string `json:"local_path,omitempty"`
	RemotePath   string `json:"remote_path,omitempty"`
	FailReason   string `json:"fail_reason,omitempty"`
	Deleted      bool   `json:"deleted,omitempty"`
	ResyncReason string `json:"resync_reason,omitempty"`
}

// SyncTaskHub 按 sync_id 分发同步任务事件给详情 stream。
type SyncTaskHub struct {
	mu          sync.RWMutex
	subscribers map[uint]map[chan SyncTaskEventPayload]struct{}
}

// GlobalSyncTaskHub 是同步任务详情 stream 使用的事件 hub。
var GlobalSyncTaskHub = NewSyncTaskHub()

// NewSyncTaskHub 创建同步任务事件 hub。
func NewSyncTaskHub() *SyncTaskHub {
	return &SyncTaskHub{subscribers: make(map[uint]map[chan SyncTaskEventPayload]struct{})}
}

// Subscribe 订阅指定同步任务事件。
func (h *SyncTaskHub) Subscribe(syncID uint, buffer int) (<-chan SyncTaskEventPayload, func()) {
	if buffer <= 0 {
		buffer = 1
	}
	ch := make(chan SyncTaskEventPayload, buffer)
	h.mu.Lock()
	if h.subscribers[syncID] == nil {
		h.subscribers[syncID] = make(map[chan SyncTaskEventPayload]struct{})
	}
	h.subscribers[syncID][ch] = struct{}{}
	h.mu.Unlock()

	var once sync.Once
	unsubscribe := func() {
		once.Do(func() {
			h.mu.Lock()
			if subscribers, ok := h.subscribers[syncID]; ok {
				if _, exists := subscribers[ch]; exists {
					delete(subscribers, ch)
					close(ch)
				}
				if len(subscribers) == 0 {
					delete(h.subscribers, syncID)
				}
			}
			h.mu.Unlock()
		})
	}
	return ch, unsubscribe
}

// Publish 发布事件给指定 sync_id 的订阅方，慢消费者会收到 resync_required 语义并被断开。
func (h *SyncTaskHub) Publish(payload SyncTaskEventPayload) {
	h.mu.Lock()
	defer h.mu.Unlock()

	subscribers := h.subscribers[payload.SyncID]
	for ch := range subscribers {
		select {
		case ch <- payload:
		default:
			resync := payload
			resync.ResyncReason = "subscriber_queue_full"
			select {
			case ch <- resync:
			default:
			}
			delete(subscribers, ch)
			close(ch)
		}
	}
	if len(subscribers) == 0 {
		delete(h.subscribers, payload.SyncID)
	}
}

var syncTaskSequences = struct {
	sync.Mutex
	values map[uint]uint64
}{values: make(map[uint]uint64)}

func nextSyncTaskSequence(syncID uint) uint64 {
	syncTaskSequences.Lock()
	defer syncTaskSequences.Unlock()
	syncTaskSequences.values[syncID]++
	return syncTaskSequences.values[syncID]
}

// BroadcastSyncTaskEvent 发布同步任务结构化事件，同时兼容全局事件 WebSocket。
func BroadcastSyncTaskEvent(eventType string, payload SyncTaskEventPayload) SyncTaskEventPayload {
	if payload.SyncID == 0 {
		return payload
	}
	if payload.EventTime == 0 {
		payload.EventTime = time.Now().Unix()
	}
	payload.Sequence = nextSyncTaskSequence(payload.SyncID)
	BroadcastEvent(eventType, payload)
	if GlobalSyncTaskHub != nil {
		GlobalSyncTaskHub.Publish(payload)
	}
	return payload
}
