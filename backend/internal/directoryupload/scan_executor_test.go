package directoryupload

import (
	"context"
	"path/filepath"
	"runtime"
	"sync/atomic"
	"testing"
	"time"

	"qmediasync/internal/models"
)

func TestScanExecutorMergesSameRuleAndRoot(t *testing.T) {
	rule := &models.DirectoryUploadRule{BaseModel: models.BaseModel{ID: 7}}
	root := filepath.Join(t.TempDir(), "Show")
	started := make(chan struct{})
	release := make(chan struct{})
	finished := make(chan struct{})
	var calls atomic.Int32

	executor := newScanExecutorWithScanFunc(1, func(ctx context.Context, _ *models.DirectoryUploadRule, _ string) (int, error) {
		calls.Add(1)
		close(started)
		select {
		case <-release:
		case <-ctx.Done():
			return 0, ctx.Err()
		}
		close(finished)
		return 0, nil
	})

	executor.Enqueue(context.Background(), scanRequest{rule: rule, root: root})
	waitForSignal(t, started, "等待首次扫描启动")
	executor.Enqueue(context.Background(), scanRequest{rule: rule, root: root})
	executor.Enqueue(context.Background(), scanRequest{rule: rule, root: filepath.Join(root, ".")})

	if got := calls.Load(); got != 1 {
		t.Fatalf("scan calls=%d，期望同一 rule/root 只启动 1 次", got)
	}
	close(release)
	waitForSignal(t, finished, "等待扫描结束")
	if got := calls.Load(); got != 1 {
		t.Fatalf("scan calls=%d，期望重复提交不补跑", got)
	}
}

func TestScanExecutorSkipsCanceledContext(t *testing.T) {
	var calls atomic.Int32
	started := make(chan struct{})
	executor := newScanExecutorWithScanFunc(1, func(context.Context, *models.DirectoryUploadRule, string) (int, error) {
		calls.Add(1)
		close(started)
		return 0, nil
	})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	executor.Enqueue(ctx, scanRequest{
		rule: &models.DirectoryUploadRule{BaseModel: models.BaseModel{ID: 8}},
		root: t.TempDir(),
	})

	select {
	case <-started:
		t.Fatal("ctx 已取消时不应启动扫描")
	case <-time.After(30 * time.Millisecond):
	}
	if got := calls.Load(); got != 0 {
		t.Fatalf("scan calls=%d，期望 0", got)
	}
}

func TestScanExecutorReplacesCanceledQueuedRequest(t *testing.T) {
	oldMaxProcs := runtime.GOMAXPROCS(1)
	defer runtime.GOMAXPROCS(oldMaxProcs)

	rule := &models.DirectoryUploadRule{BaseModel: models.BaseModel{ID: 11}}
	baseDir := t.TempDir()
	rootA := filepath.Join(baseDir, "A")
	rootK := filepath.Join(baseDir, "K")
	started := make(chan string, 2)
	releaseA := make(chan struct{})
	finished := make(chan string, 2)

	executor := newScanExecutorWithScanFunc(1, func(ctx context.Context, _ *models.DirectoryUploadRule, root string) (int, error) {
		started <- root
		if root == rootA {
			select {
			case <-releaseA:
			case <-ctx.Done():
				return 0, ctx.Err()
			}
		}
		finished <- root
		return 0, nil
	})

	executor.Enqueue(context.Background(), scanRequest{rule: rule, root: rootA})
	if got := waitForScanRoot(t, started); got != rootA {
		t.Fatalf("首次扫描 root=%s，期望 %s", got, rootA)
	}

	oldCtx, oldCancel := context.WithCancel(context.Background())
	executor.Enqueue(oldCtx, scanRequest{rule: rule, root: rootK})
	waitForQueuedScanToLeaveQueue(t, executor, rootK)

	executor.mu.Lock()
	newEnqueueReady := make(chan struct{})
	newEnqueueReturned := make(chan struct{})
	go func() {
		close(newEnqueueReady)
		executor.Enqueue(context.Background(), scanRequest{rule: rule, root: rootK})
		close(newEnqueueReturned)
	}()
	waitForSignal(t, newEnqueueReady, "等待新扫描提交开始")
	oldCancel()
	executor.mu.Unlock()
	waitForSignal(t, newEnqueueReturned, "等待新扫描提交返回")

	close(releaseA)
	if got := waitForScanRoot(t, finished); got != rootA {
		t.Fatalf("首次扫描结束 root=%s，期望 %s", got, rootA)
	}
	if got := waitForScanRoot(t, started); got != rootK {
		t.Fatalf("取消旧排队项后新扫描 root=%s，期望 %s", got, rootK)
	}
	if got := waitForScanRoot(t, finished); got != rootK {
		t.Fatalf("新扫描结束 root=%s，期望 %s", got, rootK)
	}
}

func TestScanExecutorReplacesCanceledRunningRequest(t *testing.T) {
	rule := &models.DirectoryUploadRule{BaseModel: models.BaseModel{ID: 12}}
	root := filepath.Join(t.TempDir(), "K")
	started := make(chan string, 2)
	finished := make(chan string, 2)
	releaseOld := make(chan struct{})
	oldCtx, oldCancel := context.WithCancel(context.Background())

	var calls atomic.Int32
	executor := newScanExecutorWithScanFunc(1, func(ctx context.Context, _ *models.DirectoryUploadRule, root string) (int, error) {
		call := calls.Add(1)
		started <- root
		if call == 1 {
			<-releaseOld
			return 0, ctx.Err()
		}
		finished <- root
		return 0, nil
	})

	executor.Enqueue(oldCtx, scanRequest{rule: rule, root: root})
	if got := waitForScanRoot(t, started); got != root {
		t.Fatalf("旧扫描 root=%s，期望 %s", got, root)
	}

	oldCancel()
	executor.Enqueue(context.Background(), scanRequest{rule: rule, root: root})
	close(releaseOld)

	if got := waitForScanRoot(t, started); got != root {
		t.Fatalf("新扫描 root=%s，期望 %s", got, root)
	}
	if got := waitForScanRoot(t, finished); got != root {
		t.Fatalf("新扫描结束 root=%s，期望 %s", got, root)
	}
	if got := calls.Load(); got != 2 {
		t.Fatalf("scan calls=%d，期望旧 running ctx 取消后新请求补跑", got)
	}
}

func TestScanExecutorReleasesInflightAfterScan(t *testing.T) {
	rule := &models.DirectoryUploadRule{BaseModel: models.BaseModel{ID: 9}}
	root := t.TempDir()
	started := make(chan int, 2)
	release := make(chan struct{})
	finished := make(chan int, 2)
	var calls atomic.Int32

	executor := newScanExecutorWithScanFunc(1, func(ctx context.Context, _ *models.DirectoryUploadRule, _ string) (int, error) {
		call := int(calls.Add(1))
		started <- call
		select {
		case <-release:
		case <-ctx.Done():
			return 0, ctx.Err()
		}
		finished <- call
		return 0, nil
	})

	executor.Enqueue(context.Background(), scanRequest{rule: rule, root: root})
	if got := waitForScanCall(t, started); got != 1 {
		t.Fatalf("首次扫描序号=%d，期望 1", got)
	}
	release <- struct{}{}
	if got := waitForScanCall(t, finished); got != 1 {
		t.Fatalf("首次扫描结束序号=%d，期望 1", got)
	}

	executor.Enqueue(context.Background(), scanRequest{rule: rule, root: root})
	if got := waitForScanCall(t, started); got != 2 {
		t.Fatalf("第二次扫描序号=%d，期望 inflight 释放后可再次提交", got)
	}
	release <- struct{}{}
}

func TestScanExecutorLimitsConcurrentScans(t *testing.T) {
	var active atomic.Int32
	var maxActive atomic.Int32
	started := make(chan string, 2)
	release := make(chan struct{})

	executor := newScanExecutorWithScanFunc(1, func(ctx context.Context, _ *models.DirectoryUploadRule, root string) (int, error) {
		current := active.Add(1)
		for {
			max := maxActive.Load()
			if current <= max || maxActive.CompareAndSwap(max, current) {
				break
			}
		}
		started <- root
		select {
		case <-release:
		case <-ctx.Done():
			active.Add(-1)
			return 0, ctx.Err()
		}
		active.Add(-1)
		return 0, nil
	})

	rule := &models.DirectoryUploadRule{BaseModel: models.BaseModel{ID: 10}}
	executor.Enqueue(context.Background(), scanRequest{rule: rule, root: filepath.Join(t.TempDir(), "A")})
	executor.Enqueue(context.Background(), scanRequest{rule: rule, root: filepath.Join(t.TempDir(), "B")})

	first := waitForScanRoot(t, started)
	select {
	case second := <-started:
		t.Fatalf("并发限制为 1 时第二个扫描不应同时启动，first=%s second=%s", first, second)
	case <-time.After(30 * time.Millisecond):
	}
	release <- struct{}{}
	_ = waitForScanRoot(t, started)
	release <- struct{}{}

	if got := maxActive.Load(); got > 1 {
		t.Fatalf("max active scans=%d，期望不超过 1", got)
	}
}

func waitForSignal(t *testing.T, ch <-chan struct{}, message string) {
	t.Helper()
	select {
	case <-ch:
	case <-time.After(time.Second):
		t.Fatal(message)
	}
}

func waitForScanCall(t *testing.T, ch <-chan int) int {
	t.Helper()
	select {
	case call := <-ch:
		return call
	case <-time.After(time.Second):
		t.Fatal("等待扫描调用超时")
		return 0
	}
}

func waitForScanRoot(t *testing.T, ch <-chan string) string {
	t.Helper()
	select {
	case root := <-ch:
		return root
	case <-time.After(time.Second):
		t.Fatal("等待扫描启动超时")
		return ""
	}
}

func waitForQueuedScanToLeaveQueue(t *testing.T, executor *scanExecutor, root string) {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		executor.mu.Lock()
		found := false
		for _, request := range executor.queue {
			if request.request.root == root {
				found = true
				break
			}
		}
		executor.mu.Unlock()
		if !found {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatalf("等待扫描 %s 离开队列超时", root)
}
