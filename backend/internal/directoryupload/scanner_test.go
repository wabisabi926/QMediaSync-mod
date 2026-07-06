package directoryupload

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"qmediasync/internal/db"
	"qmediasync/internal/models"
)

func TestStartRuleStartupScanAddsExistingFilesToStabilityQueue(t *testing.T) {
	setupDirectoryUploadServiceTestDB(t)
	monitorPath := t.TempDir()
	filePath := filepath.Join(monitorPath, "movie.mkv")
	writeFileWithMtime(t, filePath, []byte("movie"), time.Now())
	_, rule := createDirectoryUploadRuleForTest(t, monitorPath)
	rule.StartupScanEnabled = true

	service := NewService(ServiceOptions{})
	runtime, err := service.StartRule(context.Background(), rule)
	if err != nil {
		t.Fatalf("启动目录监控规则失败: %v", err)
	}
	defer runtime.Stop()

	got := service.PendingPaths(rule.ID)
	want := []string{filePath}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("pending paths=%v，期望启动补偿扫描发现 %v", got, want)
	}
}

func TestStartRulePollingFindsNewFiles(t *testing.T) {
	setupDirectoryUploadServiceTestDB(t)
	monitorPath := t.TempDir()
	_, rule := createDirectoryUploadRuleForTest(t, monitorPath)
	rule.WatchMode = models.DirectoryUploadWatchModePolling
	rule.StartupScanEnabled = false
	rule.RescanIntervalSeconds = 60

	service := NewService(ServiceOptions{PollInterval: 10 * time.Millisecond})
	runtime, err := service.StartRule(context.Background(), rule)
	if err != nil {
		t.Fatalf("启动 polling 目录监控规则失败: %v", err)
	}
	defer runtime.Stop()

	filePath := filepath.Join(monitorPath, "movie.mkv")
	writeFileWithMtime(t, filePath, []byte("movie"), time.Now())
	waitForPendingPath(t, service, rule.ID, filePath)
}

func TestStartRuleAutoFallsBackToPollingWhenWatcherFails(t *testing.T) {
	setupDirectoryUploadServiceTestDB(t)
	monitorPath := t.TempDir()
	_, rule := createDirectoryUploadRuleForTest(t, monitorPath)
	rule.WatchMode = models.DirectoryUploadWatchModeAuto
	rule.StartupScanEnabled = false

	service := NewService(ServiceOptions{
		PollInterval: 10 * time.Millisecond,
		WatcherFactory: func(*Service, *models.DirectoryUploadRule) (RuleWatcher, error) {
			return nil, errors.New("watcher unavailable")
		},
	})
	runtime, err := service.StartRule(context.Background(), rule)
	if err != nil {
		t.Fatalf("auto fallback 启动失败: %v", err)
	}
	defer runtime.Stop()
	if runtime.Mode != RuleRuntimeModePolling {
		t.Fatalf("runtime mode=%s，期望 fallback 到 polling", runtime.Mode)
	}

	filePath := filepath.Join(monitorPath, "episode.mp4")
	writeFileWithMtime(t, filePath, []byte("episode"), time.Now())
	waitForPendingPath(t, service, rule.ID, filePath)
}

func TestStartRuleFSNotifyFailsWhenWatcherFails(t *testing.T) {
	setupDirectoryUploadServiceTestDB(t)
	monitorPath := t.TempDir()
	_, rule := createDirectoryUploadRuleForTest(t, monitorPath)
	rule.WatchMode = models.DirectoryUploadWatchModeFSNotify
	rule.StartupScanEnabled = false

	service := NewService(ServiceOptions{
		WatcherFactory: func(*Service, *models.DirectoryUploadRule) (RuleWatcher, error) {
			return nil, errors.New("watcher unavailable")
		},
	})
	if _, err := service.StartRule(context.Background(), rule); err == nil {
		t.Fatal("fsnotify 模式下 watcher 初始化失败应直接报错")
	}
}

func TestStartRuleRejectsLegacyWatcherMode(t *testing.T) {
	setupDirectoryUploadServiceTestDB(t)
	monitorPath := t.TempDir()
	_, rule := createDirectoryUploadRuleForTest(t, monitorPath)
	rule.WatchMode = models.DirectoryUploadWatchMode("watcher")
	rule.StartupScanEnabled = false

	service := NewService(ServiceOptions{})
	if _, err := service.StartRule(context.Background(), rule); err == nil {
		t.Fatal("旧 watcher 监控模式不应继续兼容")
	}
}

func TestStartRunsProcessedCleanupOnStartupAndInterval(t *testing.T) {
	setupDirectoryUploadServiceTestDB(t)
	clock := &fakeClock{now: time.Unix(200_000, 0)}
	oldPath := filepath.Join(t.TempDir(), "missing-startup.mkv")
	startupRecord := &models.DirectoryUploadProcessedFile{
		SourceKey:         "cleanup-startup",
		LocalFullPath:     oldPath,
		SourceFingerprint: "v1:1:1",
		Result:            models.DirectoryUploadProcessedResultUploaded,
		ProcessedAt:       clock.Now().Add(-48 * time.Hour).Unix(),
		LastSeenAt:        clock.Now().Add(-48 * time.Hour).Unix(),
	}
	if err := db.Db.Create(startupRecord).Error; err != nil {
		t.Fatalf("创建启动清理 processed 记录失败: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	service := NewService(ServiceOptions{
		Now:                       clock.Now,
		ProcessedCleanupInterval:  10 * time.Millisecond,
		ProcessedMissingSourceTTL: 24 * time.Hour,
	})
	if err := service.Start(ctx); err != nil {
		cancel()
		t.Fatalf("启动目录上传服务失败: %v", err)
	}
	defer func() {
		cancel()
		service.Stop()
	}()

	waitForProcessedCount(t, 0)

	intervalPath := filepath.Join(t.TempDir(), "missing-interval.mkv")
	intervalRecord := &models.DirectoryUploadProcessedFile{
		SourceKey:         "cleanup-interval",
		LocalFullPath:     intervalPath,
		SourceFingerprint: "v1:2:2",
		Result:            models.DirectoryUploadProcessedResultRemoteExists,
		ProcessedAt:       clock.Now().Add(-48 * time.Hour).Unix(),
		LastSeenAt:        clock.Now().Add(-48 * time.Hour).Unix(),
	}
	if err := db.Db.Create(intervalRecord).Error; err != nil {
		t.Fatalf("创建周期清理 processed 记录失败: %v", err)
	}
	waitForProcessedCount(t, 0)
	if _, err := os.Stat(intervalPath); !os.IsNotExist(err) {
		t.Fatalf("测试源文件应保持缺失状态: %s err=%v", intervalPath, err)
	}
}

func TestPollingIntervalUsesBuiltInDefault(t *testing.T) {
	rule := &models.DirectoryUploadRule{RescanIntervalSeconds: 999}
	service := NewService(ServiceOptions{})

	if got := service.pollingInterval(rule); got != 30*time.Second {
		t.Fatalf("polling interval = %s，期望使用内置 30s", got)
	}
}

func TestPollingIntervalAllowsTestOverride(t *testing.T) {
	rule := &models.DirectoryUploadRule{RescanIntervalSeconds: 999}
	service := NewService(ServiceOptions{PollInterval: 10 * time.Millisecond})

	if got := service.pollingInterval(rule); got != 10*time.Millisecond {
		t.Fatalf("polling interval = %s，期望测试注入优先", got)
	}
}

func TestStabilityIntervalUsesBuiltInDefault(t *testing.T) {
	rule := &models.DirectoryUploadRule{StabilityCheckIntervalSeconds: 999}
	service := NewService(ServiceOptions{})

	if got := service.stabilityInterval(rule); got != 2*time.Second {
		t.Fatalf("stability interval = %s，期望使用内置 2s", got)
	}
}

func TestStabilityIntervalAllowsTestOverride(t *testing.T) {
	rule := &models.DirectoryUploadRule{StabilityCheckIntervalSeconds: 999}
	service := NewService(ServiceOptions{StabilityCheckInterval: 10 * time.Millisecond})

	if got := service.stabilityInterval(rule); got != 10*time.Millisecond {
		t.Fatalf("stability interval = %s，期望测试注入优先", got)
	}
}

func waitForPendingPath(t *testing.T, service *Service, ruleID uint, filePath string) {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		for _, pending := range service.PendingPaths(ruleID) {
			if pending == filePath {
				return
			}
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("等待 pending path 超时: %s, got=%v", filePath, service.PendingPaths(ruleID))
}

func waitForProcessedCount(t *testing.T, want int64) {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		var got int64
		if err := db.Db.Model(&models.DirectoryUploadProcessedFile{}).Count(&got).Error; err != nil {
			t.Fatalf("统计 processed 记录失败: %v", err)
		}
		if got == want {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	var got int64
	if err := db.Db.Model(&models.DirectoryUploadProcessedFile{}).Count(&got).Error; err != nil {
		t.Fatalf("统计 processed 记录失败: %v", err)
	}
	t.Fatalf("等待 processed 记录数量超时: got=%d want=%d", got, want)
}
