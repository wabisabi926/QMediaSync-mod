package directoryupload

import (
	"context"
	"errors"
	"path/filepath"
	"reflect"
	"testing"
	"time"

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
