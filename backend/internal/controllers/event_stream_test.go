package controllers

import (
	"bufio"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"qmediasync/internal/realtime"

	"github.com/gin-gonic/gin"
)

func TestEventStreamSendsSSEEventAndCleansUpSubscription(t *testing.T) {
	oldHub := realtime.GlobalEventHub
	oldLifecycle := realtime.GlobalLifecycle
	realtime.GlobalEventHub = realtime.NewEventHub()
	realtime.GlobalLifecycle = realtime.NewLifecycle()
	t.Cleanup(func() {
		realtime.GlobalEventHub = oldHub
		realtime.GlobalLifecycle = oldLifecycle
	})

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/events/stream", EventStream)
	server := httptest.NewServer(router)
	defer server.Close()

	response, err := http.Get(server.URL + "/events/stream")
	if err != nil {
		t.Fatalf("建立 SSE 请求失败：%v", err)
	}
	defer response.Body.Close()
	if !strings.HasPrefix(response.Header.Get("Content-Type"), "text/event-stream") {
		t.Fatalf("Content-Type = %q，期望 SSE", response.Header.Get("Content-Type"))
	}

	realtime.BroadcastEvent(realtime.EventUploadQueueChanged, map[string]string{"reason": "test"})
	reader := bufio.NewReader(response.Body)
	var eventFrame strings.Builder
	for {
		line, readErr := reader.ReadString('\n')
		if readErr != nil {
			t.Fatalf("读取 SSE 事件失败：%v", readErr)
		}
		eventFrame.WriteString(line)
		if line != "\n" {
			continue
		}
		if strings.Contains(eventFrame.String(), "event:upload_queue_changed") {
			break
		}
		eventFrame.Reset()
	}
	if !strings.Contains(eventFrame.String(), `"event_type":"upload_queue_changed"`) {
		t.Fatalf("SSE data 未保留完整全局事件: %s", eventFrame.String())
	}

	_ = response.Body.Close()
	deadline := time.After(time.Second)
	for realtime.GlobalEventHub.ClientCount() != 0 {
		select {
		case <-deadline:
			t.Fatal("请求结束后全局事件订阅未注销")
		default:
			time.Sleep(time.Millisecond)
		}
	}
}

func TestEventStreamDoesNotSubscribeWhenLifecycleAlreadyStopped(t *testing.T) {
	oldHub := realtime.GlobalEventHub
	oldLifecycle := realtime.GlobalLifecycle
	realtime.GlobalEventHub = realtime.NewEventHub()
	realtime.GlobalLifecycle = realtime.NewLifecycle()
	realtime.GlobalLifecycle.Shutdown()
	t.Cleanup(func() {
		realtime.GlobalEventHub = oldHub
		realtime.GlobalLifecycle = oldLifecycle
	})

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodGet, "/events/stream", nil)

	EventStream(context)

	if realtime.GlobalEventHub.ClientCount() != 0 {
		t.Fatal("Lifecycle 已停止时不应建立全局事件订阅")
	}
	if recorder.Body.Len() != 0 {
		t.Fatalf("Lifecycle 已停止时不应写入 SSE 首帧: %q", recorder.Body.String())
	}
}

func TestSSEFrameDoesNotDelayLifecycleShutdown(t *testing.T) {
	oldLifecycle := realtime.GlobalLifecycle
	realtime.GlobalLifecycle = realtime.NewLifecycle()
	t.Cleanup(func() {
		realtime.GlobalLifecycle = oldLifecycle
	})

	writeStarted := make(chan struct{})
	releaseWrite := make(chan struct{})
	writeDone := make(chan struct{})
	go func() {
		defer close(writeDone)
		_ = writeSSEFrame(func() error {
			close(writeStarted)
			<-releaseWrite
			return nil
		})
	}()
	<-writeStarted

	shutdownDone := make(chan struct{})
	go func() {
		realtime.GlobalLifecycle.Shutdown()
		close(shutdownDone)
	}()

	shutdownBlocked := false
	select {
	case <-shutdownDone:
	case <-time.After(100 * time.Millisecond):
		shutdownBlocked = true
	}

	close(releaseWrite)
	<-writeDone
	<-shutdownDone
	if shutdownBlocked {
		t.Fatal("单个 SSE 帧写入不应阻塞 Lifecycle Shutdown")
	}
}

func TestEventStreamStopsSubscribedConnectionOnLifecycleShutdown(t *testing.T) {
	oldHub := realtime.GlobalEventHub
	oldLifecycle := realtime.GlobalLifecycle
	realtime.GlobalEventHub = realtime.NewEventHub()
	realtime.GlobalLifecycle = realtime.NewLifecycle()
	t.Cleanup(func() {
		realtime.GlobalEventHub = oldHub
		realtime.GlobalLifecycle = oldLifecycle
	})

	router := gin.New()
	router.GET("/events/stream", EventStream)
	server := httptest.NewServer(router)
	defer server.Close()

	response, err := server.Client().Get(server.URL + "/events/stream")
	if err != nil {
		t.Fatalf("建立 SSE 请求失败: %v", err)
	}
	defer response.Body.Close()
	reader := bufio.NewReader(response.Body)
	connected := readSSEFrame(t, reader)
	if !strings.Contains(connected, ": connected") {
		t.Fatalf("首帧应为 connected 注释，frame = %q", connected)
	}
	if realtime.GlobalEventHub.ClientCount() != 1 {
		t.Fatalf("当前订阅数 = %d，期望 1", realtime.GlobalEventHub.ClientCount())
	}

	realtime.GlobalLifecycle.Shutdown()
	readResult := make(chan error, 1)
	go func() {
		_, err := reader.ReadString('\n')
		readResult <- err
	}()
	select {
	case err := <-readResult:
		if err == nil {
			t.Fatal("Lifecycle 停止后 SSE 响应应结束")
		}
	case <-time.After(time.Second):
		t.Fatal("Lifecycle 停止后 SSE 响应未及时结束")
	}

	deadline := time.After(time.Second)
	for realtime.GlobalEventHub.ClientCount() != 0 {
		select {
		case <-deadline:
			t.Fatal("Lifecycle 停止后全局事件订阅未注销")
		default:
			time.Sleep(time.Millisecond)
		}
	}
}
