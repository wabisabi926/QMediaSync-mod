package v115open

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"resty.dev/v3"
)

func TestQueueExecutorStopDrainsBufferedRequests(t *testing.T) {
	ensureOpenAPITestLoggers()

	firstStarted := make(chan struct{})
	releaseFirst := make(chan struct{})
	var first sync.Once
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		first.Do(func() {
			close(firstStarted)
			<-releaseFirst
		})
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"state":true,"code":0,"data":{}}`))
	}))
	t.Cleanup(server.Close)

	executor := NewQueueExecutor(100, 6000, 60000)
	executor.workerCount = 1
	executor.Start()
	t.Cleanup(executor.Stop)

	client := resty.New()
	const requestCount = 50
	responseChans := make([]chan *RequestResponse, 0, requestCount)
	for range requestCount {
		req := client.R()
		req.Method = http.MethodGet
		respChan := make(chan *RequestResponse, 1)
		responseChans = append(responseChans, respChan)
		executor.EnqueueRequest(&QueuedRequest{
			URL:             server.URL,
			Method:          http.MethodGet,
			Request:         req,
			BypassRateLimit: true,
			ResponseChan:    respChan,
			CreatedAt:       time.Now(),
			Ctx:             context.Background(),
		})
	}

	select {
	case <-firstStarted:
	case <-time.After(3 * time.Second):
		t.Fatal("第一个请求未开始处理")
	}

	stopDone := make(chan struct{})
	go func() {
		executor.Stop()
		close(stopDone)
	}()

	close(releaseFirst)

	select {
	case <-stopDone:
	case <-time.After(3 * time.Second):
		t.Fatal("Stop() 未等待队列请求处理完成")
	}

	for i, respChan := range responseChans {
		select {
		case resp := <-respChan:
			if resp == nil {
				t.Fatalf("请求 %d 响应为空", i)
			}
			if resp.Error != nil {
				t.Fatalf("请求 %d 返回错误: %v", i, resp.Error)
			}
		case <-time.After(500 * time.Millisecond):
			t.Fatalf("请求 %d 未收到响应，Stop() 丢弃了已入队请求", i)
		}
	}
}

func TestQueueExecutorStopRejectsNewRequestsWhileQueueSendBlocked(t *testing.T) {
	ensureOpenAPITestLoggers()

	firstStarted := make(chan struct{})
	releaseFirst := make(chan struct{})
	var first sync.Once
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		first.Do(func() {
			close(firstStarted)
			<-releaseFirst
		})
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"state":true,"code":0,"data":{}}`))
	}))
	t.Cleanup(server.Close)

	executor := NewQueueExecutor(100, 6000, 60000)
	executor.workerCount = 1
	executor.Start()
	t.Cleanup(executor.Stop)

	client := resty.New()
	enqueue := func() chan *RequestResponse {
		req := client.R()
		req.Method = http.MethodGet
		respChan := make(chan *RequestResponse, 1)
		executor.EnqueueRequest(&QueuedRequest{
			URL:             server.URL,
			Method:          http.MethodGet,
			Request:         req,
			BypassRateLimit: true,
			ResponseChan:    respChan,
			CreatedAt:       time.Now(),
			Ctx:             context.Background(),
		})
		return respChan
	}

	enqueue()
	select {
	case <-firstStarted:
	case <-time.After(3 * time.Second):
		t.Fatal("第一个请求未开始处理")
	}

	for range 100 {
		enqueue()
	}

	blockedEnqueueDone := make(chan struct{})
	go func() {
		enqueue()
		close(blockedEnqueueDone)
	}()

	select {
	case <-blockedEnqueueDone:
		t.Fatal("额外入队请求未被满队列阻塞")
	case <-time.After(100 * time.Millisecond):
	}

	stopDone := make(chan struct{})
	go func() {
		executor.Stop()
		close(stopDone)
	}()

	stoppedStateVisible := make(chan struct{})
	go func() {
		for {
			executor.RLock()
			stopped := !executor.running && executor.requestQueue == nil
			executor.RUnlock()
			if stopped {
				close(stoppedStateVisible)
				return
			}
			time.Sleep(10 * time.Millisecond)
		}
	}()

	select {
	case <-stoppedStateVisible:
	case <-time.After(500 * time.Millisecond):
		close(releaseFirst)
		<-stopDone
		t.Fatal("Stop() 未能在排空旧队列前切换执行器状态")
	}

	rejectedRespChan := make(chan *RequestResponse, 1)
	enqueueReturned := make(chan struct{})
	go func() {
		req := client.R()
		req.Method = http.MethodGet
		executor.EnqueueRequest(&QueuedRequest{
			URL:             server.URL,
			Method:          http.MethodGet,
			Request:         req,
			BypassRateLimit: true,
			ResponseChan:    rejectedRespChan,
			CreatedAt:       time.Now(),
			Ctx:             context.Background(),
		})
		close(enqueueReturned)
	}()

	select {
	case <-enqueueReturned:
	case <-time.After(500 * time.Millisecond):
		close(releaseFirst)
		<-stopDone
		t.Fatal("Stop() 排空旧队列时，新入队请求被执行器锁阻塞")
	}

	select {
	case resp := <-rejectedRespChan:
		if resp == nil || resp.Error == nil {
			close(releaseFirst)
			<-stopDone
			t.Fatalf("新入队请求应返回未启动错误，实际响应: %#v", resp)
		}
	case <-time.After(500 * time.Millisecond):
		close(releaseFirst)
		<-stopDone
		t.Fatal("新入队请求未收到未启动错误")
	}

	close(releaseFirst)

	select {
	case <-stopDone:
	case <-time.After(3 * time.Second):
		t.Fatal("Stop() 未在释放阻塞请求后完成")
	}
}
