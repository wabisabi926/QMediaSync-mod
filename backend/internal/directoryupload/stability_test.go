package directoryupload

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"qmediasync/internal/models"
)

type fakeClock struct {
	now time.Time
}

func (c *fakeClock) Now() time.Time {
	return c.now
}

func (c *fakeClock) Add(d time.Duration) {
	c.now = c.now.Add(d)
}

func TestStabilityQueueWaitsForStableFile(t *testing.T) {
	clock := &fakeClock{now: time.Unix(100, 0)}
	dir := t.TempDir()
	path := filepath.Join(dir, "movie.mkv")
	writeFileWithMtime(t, path, []byte("one"), clock.Now())

	queue := NewStabilityQueue(StabilityQueueOptions{Now: clock.Now})
	rule := &models.DirectoryUploadRule{
		BaseModel:              models.BaseModel{ID: 1},
		StabilitySeconds:       2,
		StabilityRequiredCount: 2,
	}
	queue.Track(rule.ID, path)

	ready, err := queue.Check(rule)
	if err != nil {
		t.Fatalf("首次稳定性检查失败: %v", err)
	}
	if len(ready) != 0 {
		t.Fatalf("首次检查不应入队，got=%v", ready)
	}

	clock.Add(time.Second)
	writeFileWithMtime(t, path, []byte("changed"), clock.Now())
	ready, err = queue.Check(rule)
	if err != nil {
		t.Fatalf("变化后稳定性检查失败: %v", err)
	}
	if len(ready) != 0 {
		t.Fatalf("文件大小变化后不应入队，got=%v", ready)
	}

	clock.Add(2 * time.Second)
	ready, err = queue.Check(rule)
	if err != nil {
		t.Fatalf("稳定窗口首次检查失败: %v", err)
	}
	if len(ready) != 0 {
		t.Fatalf("稳定计数未满足时不应入队，got=%v", ready)
	}

	ready, err = queue.Check(rule)
	if err != nil {
		t.Fatalf("稳定窗口第二次检查失败: %v", err)
	}
	if len(ready) != 1 || ready[0].Path != path {
		t.Fatalf("ready=%v，期望稳定文件 %s", ready, path)
	}
}

func TestStabilityQueueResetsWhenSignatureChanges(t *testing.T) {
	clock := &fakeClock{now: time.Unix(200, 0)}
	dir := t.TempDir()
	path := filepath.Join(dir, "episode.mkv")
	writeFileWithMtime(t, path, []byte("one"), clock.Now())

	queue := NewStabilityQueue(StabilityQueueOptions{Now: clock.Now})
	rule := &models.DirectoryUploadRule{
		BaseModel:              models.BaseModel{ID: 2},
		StabilitySeconds:       0,
		StabilityRequiredCount: 2,
	}
	queue.Track(rule.ID, path)

	if ready, err := queue.Check(rule); err != nil || len(ready) != 0 {
		t.Fatalf("首次检查 ready=%v err=%v，期望未稳定", ready, err)
	}
	clock.Add(time.Second)
	writeFileWithMtime(t, path, []byte("one"), clock.Now())
	if ready, err := queue.Check(rule); err != nil || len(ready) != 0 {
		t.Fatalf("mtime 变化后 ready=%v err=%v，期望重置稳定计数", ready, err)
	}
	if ready, err := queue.Check(rule); err != nil || len(ready) != 1 {
		t.Fatalf("签名连续不变后 ready=%v err=%v，期望入队", ready, err)
	}
}

func writeFileWithMtime(t *testing.T, path string, data []byte, mtime time.Time) {
	t.Helper()
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("写入测试文件失败: %v", err)
	}
	if err := os.Chtimes(path, mtime, mtime); err != nil {
		t.Fatalf("设置测试文件 mtime 失败: %v", err)
	}
}
