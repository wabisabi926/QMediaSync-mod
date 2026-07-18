package realtime

import (
	"context"
)

// Lifecycle 管理实时流的进程级停机信号。
type Lifecycle struct {
	ctx    context.Context
	cancel context.CancelFunc
}

// GlobalLifecycle 是实时流共用的进程级生命周期。
var GlobalLifecycle = NewLifecycle()

// NewLifecycle 创建实时流生命周期。
func NewLifecycle() *Lifecycle {
	ctx, cancel := context.WithCancel(context.Background())
	return &Lifecycle{ctx: ctx, cancel: cancel}
}

// Done 返回实时流停机通知。
func (l *Lifecycle) Done() <-chan struct{} {
	return l.ctx.Done()
}

// IsStopped 返回实时流生命周期是否已进入停机状态。
func (l *Lifecycle) IsStopped() bool {
	select {
	case <-l.ctx.Done():
		return true
	default:
		return false
	}
}

// Shutdown 停止所有实时流，重复调用安全。
func (l *Lifecycle) Shutdown() {
	l.cancel()
}

// StreamContext 将请求和进程停机信号合并为流式请求 context。
func (l *Lifecycle) StreamContext(parent context.Context) (context.Context, func()) {
	ctx, cancel := context.WithCancel(parent)
	stop := context.AfterFunc(l.ctx, cancel)
	if l.IsStopped() {
		cancel()
	}

	return ctx, func() {
		stop()
		cancel()
	}
}
