package websocket

import (
	"encoding/json"
	"testing"
	"time"
)

func readBroadcastEvent(t *testing.T, hub *EventHub) WSEvent {
	t.Helper()
	select {
	case msg := <-hub.broadcast:
		var event WSEvent
		if err := json.Unmarshal(msg, &event); err != nil {
			t.Fatalf("解析 WebSocket 事件失败: %v", err)
		}
		return event
	default:
		t.Fatal("未收到广播事件")
		return WSEvent{}
	}
}

func TestBroadcastQueueStatusChanged(t *testing.T) {
	oldHub := GlobalEventHub
	t.Cleanup(func() { GlobalEventHub = oldHub })
	GlobalEventHub = NewEventHub()

	BroadcastQueueStatusChanged(EventUploadQueueStatusChanged, true)
	event := readBroadcastEvent(t, GlobalEventHub)

	if event.EventType != EventUploadQueueStatusChanged {
		t.Fatalf("event_type = %s，期望 %s", event.EventType, EventUploadQueueStatusChanged)
	}
	data, ok := event.Data.(map[string]any)
	if !ok {
		t.Fatalf("data 类型 = %T，期望 map[string]any", event.Data)
	}
	if data["running"] != true {
		t.Fatalf("running = %v，期望 true", data["running"])
	}
}

func TestBroadcastQueueChanged(t *testing.T) {
	oldHub := GlobalEventHub
	t.Cleanup(func() { GlobalEventHub = oldHub })
	GlobalEventHub = NewEventHub()

	BroadcastQueueChanged(EventDownloadQueueChanged, QueueChangedPayload{
		TaskID: 88,
		Status: 1,
		Source: "strm_sync",
	})
	event := readBroadcastEvent(t, GlobalEventHub)

	if event.EventType != EventDownloadQueueChanged {
		t.Fatalf("event_type = %s，期望 %s", event.EventType, EventDownloadQueueChanged)
	}
	data, ok := event.Data.(map[string]any)
	if !ok {
		t.Fatalf("data 类型 = %T，期望 map[string]any", event.Data)
	}
	if data["task_id"] != float64(88) {
		t.Fatalf("task_id = %v，期望 88", data["task_id"])
	}
	if data["status"] != float64(1) {
		t.Fatalf("status = %v，期望 1", data["status"])
	}
}

func TestTryBroadcastEventDoesNotBlockWhenBroadcastChannelFull(t *testing.T) {
	oldHub := GlobalEventHub
	t.Cleanup(func() { GlobalEventHub = oldHub })
	GlobalEventHub = NewEventHub()
	for i := 0; i < cap(GlobalEventHub.broadcast); i++ {
		GlobalEventHub.broadcast <- []byte("{}")
	}

	done := make(chan bool, 1)
	go func() {
		done <- TryBroadcastEvent(EventStrmSyncTaskQueued, map[string]any{"sync_path_id": 1})
	}()

	select {
	case ok := <-done:
		if ok {
			t.Fatal("广播通道已满时 TryBroadcastEvent 应返回 false")
		}
	case <-time.After(200 * time.Millisecond):
		t.Fatal("广播通道已满时 TryBroadcastEvent 不应阻塞")
	}
}
