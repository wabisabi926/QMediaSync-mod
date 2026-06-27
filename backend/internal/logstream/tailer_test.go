package logstream

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestManagerSharesTailerForSamePath(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sync.log")
	if err := os.WriteFile(path, []byte(""), 0o644); err != nil {
		t.Fatal(err)
	}

	manager := NewManager()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	first, firstClose, err := manager.Subscribe(ctx, path, 0, 8)
	if err != nil {
		t.Fatalf("订阅 first 失败：%v", err)
	}
	defer firstClose()
	second, secondClose, err := manager.Subscribe(ctx, path, 0, 8)
	if err != nil {
		t.Fatalf("订阅 second 失败：%v", err)
	}
	defer secondClose()

	if manager.TailerCount() != 1 {
		t.Fatalf("tailer count = %d，期望 1", manager.TailerCount())
	}

	file, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		t.Fatal(err)
	}
	_, _ = file.WriteString("2025/11/29 12:33:11.000001 [INFO] shared\n")
	_ = file.Close()

	assertLine := func(name string, ch <-chan Message) {
		t.Helper()
		select {
		case msg := <-ch:
			if msg.Entry.Message != "shared" {
				t.Fatalf("%s message = %s，期望 shared", name, msg.Entry.Message)
			}
		case <-time.After(2 * time.Second):
			t.Fatalf("%s 未收到日志", name)
		}
	}
	assertLine("first", first)
	assertLine("second", second)
}
