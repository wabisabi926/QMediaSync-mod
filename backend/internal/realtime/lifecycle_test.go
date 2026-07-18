package realtime

import (
	"context"
	"testing"
	"time"
)

func TestLifecycleStreamContextCancelsWithLifecycle(t *testing.T) {
	lifecycle := NewLifecycle()
	ctx, cleanup := lifecycle.StreamContext(context.Background())
	defer cleanup()

	lifecycle.Shutdown()

	select {
	case <-ctx.Done():
	case <-time.After(time.Second):
		t.Fatal("生命周期停止后 stream context 未取消")
	}
}

func TestLifecycleStreamContextCancelsWithRequest(t *testing.T) {
	lifecycle := NewLifecycle()
	parent, cancel := context.WithCancel(context.Background())
	ctx, cleanup := lifecycle.StreamContext(parent)
	defer cleanup()

	cancel()

	select {
	case <-ctx.Done():
	case <-time.After(time.Second):
		t.Fatal("请求停止后 stream context 未取消")
	}
}

func TestLifecycleStreamContextIsCancelledImmediatelyWhenAlreadyStopped(t *testing.T) {
	lifecycle := NewLifecycle()
	lifecycle.Shutdown()

	ctx, cleanup := lifecycle.StreamContext(context.Background())
	defer cleanup()

	if err := ctx.Err(); err != context.Canceled {
		t.Fatalf("已停止的 Lifecycle 创建 stream context 后 err = %v，期望 %v", err, context.Canceled)
	}
}

func TestLifecycleShutdownIsIdempotent(t *testing.T) {
	lifecycle := NewLifecycle()
	lifecycle.Shutdown()
	lifecycle.Shutdown()

	select {
	case <-lifecycle.Done():
	case <-time.After(time.Second):
		t.Fatal("重复 Shutdown 后 Done 未关闭")
	}
}
