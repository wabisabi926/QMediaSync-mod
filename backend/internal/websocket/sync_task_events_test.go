package websocket

import (
	"encoding/json"
	"testing"
	"time"
)

func TestBroadcastSyncTaskEventAssignsPerTaskSequenceAndBroadcasts(t *testing.T) {
	oldHub := GlobalEventHub
	oldSyncHub := GlobalSyncTaskHub
	t.Cleanup(func() {
		GlobalEventHub = oldHub
		GlobalSyncTaskHub = oldSyncHub
	})

	GlobalEventHub = NewEventHub()
	GlobalSyncTaskHub = NewSyncTaskHub()

	first := BroadcastSyncTaskEvent(EventSyncTaskUpdated, SyncTaskEventPayload{
		SyncID:     10,
		SyncPathID: 2,
		Status:     1,
		SubStatus:  1,
		LogPath:    "libs/sync_10.log",
	})
	second := BroadcastSyncTaskEvent(EventSyncTaskUpdated, SyncTaskEventPayload{
		SyncID:     10,
		SyncPathID: 2,
		Status:     1,
		SubStatus:  2,
		LogPath:    "libs/sync_10.log",
	})

	if first.Sequence != 1 {
		t.Fatalf("first sequence = %d，期望 1", first.Sequence)
	}
	if second.Sequence != 2 {
		t.Fatalf("second sequence = %d，期望 2", second.Sequence)
	}

	raw := <-GlobalEventHub.broadcast
	var event WSEvent
	if err := json.Unmarshal(raw, &event); err != nil {
		t.Fatalf("解析全局事件失败：%v", err)
	}
	if event.EventType != EventSyncTaskUpdated {
		t.Fatalf("event_type = %s，期望 %s", event.EventType, EventSyncTaskUpdated)
	}
}

func TestSyncTaskHubSubscriberReceivesOnlyMatchingTask(t *testing.T) {
	hub := NewSyncTaskHub()
	ch, unsubscribe := hub.Subscribe(10, 8)
	defer unsubscribe()

	hub.Publish(SyncTaskEventPayload{SyncID: 11, Sequence: 1})
	hub.Publish(SyncTaskEventPayload{SyncID: 10, Sequence: 2})

	select {
	case payload := <-ch:
		if payload.SyncID != 10 || payload.Sequence != 2 {
			t.Fatalf("payload = %+v，期望 sync_id=10 sequence=2", payload)
		}
	case <-time.After(time.Second):
		t.Fatal("未收到匹配任务事件")
	}
}
