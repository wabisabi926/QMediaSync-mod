package controllers

import (
	"os"
	"strings"
	"testing"
)

func TestEmbyWebhook新增事件文案指向条目同步(t *testing.T) {
	source := readSourceForBoundaryTest(t, "emby.go")

	if strings.Contains(source, "同步一次 Emby 媒体库") {
		t.Fatal("Webhook 新增事件不应描述为同步 Emby 媒体库，应描述为同步 Emby 条目到本地")
	}
	if !strings.Contains(source, "同步 Emby 条目到本地") {
		t.Fatal("Webhook 新增事件应明确描述为同步 Emby 条目到本地")
	}
}

func TestEmby两条链路不互相调用(t *testing.T) {
	refreshSource := readSourceForBoundaryTest(t, "../models/emby_library_refresh_task.go")
	if strings.Contains(refreshSource, "PerformEmbySync") ||
		strings.Contains(refreshSource, "IncrementalSyncEmbyMediaItems") {
		t.Fatal("链路 A 刷新 Emby 扫描库不应调用链路 B 条目同步")
	}

	syncSource := readSourceForBoundaryTest(t, "../emby/emby.go")
	if strings.Contains(syncSource, "RefreshLibrary(") {
		t.Fatal("链路 B 同步 Emby 条目到本地不应调用 RefreshLibrary")
	}
}

func readSourceForBoundaryTest(t *testing.T, path string) string {
	t.Helper()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("读取源码文件 %s 失败: %v", path, err)
	}
	return string(data)
}
