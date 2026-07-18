package controllers

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"qmediasync/internal/helpers"
	"qmediasync/internal/realtime"

	"github.com/gin-gonic/gin"
)

const (
	sseKeepaliveInterval = 15 * time.Second
	sseWriteTimeout      = 30 * time.Second
)

// EventStream 推送全局业务 SSE 事件。
func EventStream(c *gin.Context) {
	streamCtx, cleanup := realtime.GlobalLifecycle.StreamContext(c.Request.Context())
	defer cleanup()
	if isSSEStreamStopped(streamCtx) {
		return
	}

	events, unsubscribe := realtime.GlobalEventHub.Subscribe(256)
	defer unsubscribe()
	if isSSEStreamStopped(streamCtx) {
		return
	}
	setSSEHeaders(c)
	if err := writeSSEComment(c, "connected"); err != nil {
		if isSSEStreamStopError(err) {
			return
		}
		helpers.AppLogger.Errorf("全局 SSE 建连失败: %v", err)
		return
	}

	keepalive := time.NewTicker(sseKeepaliveInterval)
	defer keepalive.Stop()
	for {
		select {
		case <-streamCtx.Done():
			return
		case <-keepalive.C:
			if isSSEStreamStopped(streamCtx) {
				return
			}
			if err := writeSSEComment(c, "keepalive"); err != nil {
				if isSSEStreamStopError(err) {
					return
				}
				helpers.AppLogger.Errorf("写入全局 SSE 心跳失败: %v", err)
				return
			}
		case event, ok := <-events:
			if !ok {
				return
			}
			if isSSEStreamStopped(streamCtx) {
				return
			}
			if err := writeSSEEvent(c, event.EventType, event); err != nil {
				if isSSEStreamStopError(err) {
					return
				}
				helpers.AppLogger.Errorf("写入全局 SSE 事件失败: %v", err)
				return
			}
		}
	}
}

func setSSEHeaders(c *gin.Context) {
	c.Header("Content-Type", "text/event-stream; charset=utf-8")
	c.Header("Cache-Control", "no-cache")
	c.Header("X-Accel-Buffering", "no")
}

func setSSEWriteDeadline(c *gin.Context) error {
	return http.NewResponseController(c.Writer).SetWriteDeadline(time.Now().Add(sseWriteTimeout))
}

func writeSSEComment(c *gin.Context, comment string) error {
	return writeSSEFrame(func() error {
		if err := setSSEWriteDeadline(c); err != nil {
			return err
		}
		if _, err := c.Writer.WriteString(fmt.Sprintf(": %s\n\n", comment)); err != nil {
			return err
		}
		c.Writer.Flush()
		return nil
	})
}

func writeSSEEvent(c *gin.Context, eventType string, data any) error {
	return writeSSEFrame(func() error {
		if err := setSSEWriteDeadline(c); err != nil {
			return err
		}
		c.SSEvent(eventType, data)
		c.Writer.Flush()
		return nil
	})
}

var errSSEStreamStopped = errors.New("SSE stream lifecycle stopped")

func writeSSEFrame(write func() error) error {
	if realtime.GlobalLifecycle.IsStopped() {
		return errSSEStreamStopped
	}
	return write()
}

func isSSEStreamStopError(err error) bool {
	return errors.Is(err, errSSEStreamStopped)
}

func isSSEStreamStopped(streamCtx context.Context) bool {
	return streamCtx.Err() != nil || realtime.GlobalLifecycle.IsStopped()
}
