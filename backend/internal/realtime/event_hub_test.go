package realtime

import (
	"testing"
	"time"
)

func TestEventHubPublishesAndUnsubscribes(t *testing.T) {
	hub := NewEventHub()
	events, unsubscribe := hub.Subscribe(1)

	hub.Publish(RealtimeEvent{EventType: EventUploadQueueChanged, Data: "changed"})

	select {
	case event := <-events:
		if event.EventType != EventUploadQueueChanged || event.Data != "changed" {
			t.Fatalf("收到错误事件: %#v", event)
		}
	case <-time.After(time.Second):
		t.Fatal("未收到已发布事件")
	}

	unsubscribe()
	if _, ok := <-events; ok {
		t.Fatal("注销后订阅 channel 未关闭")
	}
}

func TestEventHubClosesSlowSubscriberWithoutBlockingPublisher(t *testing.T) {
	hub := NewEventHub()
	slowEvents, _ := hub.Subscribe(1)

	hub.Publish(RealtimeEvent{EventType: EventUploadQueueChanged})
	hub.Publish(RealtimeEvent{EventType: EventUploadQueueChanged})

	select {
	case _, ok := <-slowEvents:
		if ok {
			_, ok = <-slowEvents
		}
		if ok {
			t.Fatal("慢订阅者未被关闭")
		}
	case <-time.After(time.Second):
		t.Fatal("发布被慢订阅者阻塞")
	}
}
