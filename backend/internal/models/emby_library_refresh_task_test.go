package models

import (
	"Q115-STRM/internal/db"
	"reflect"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func setupEmbyRefreshTestDB(t *testing.T) {
	t.Helper()
	testDb, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("打开测试数据库失败: %v", err)
	}
	db.Db = testDb
	if err := db.Db.AutoMigrate(
		&EmbyConfig{},
		&EmbyLibrarySyncPath{},
		&EmbyLibraryRefreshTask{},
		&SyncFile{},
		&DbDownloadTask{},
	); err != nil {
		t.Fatalf("迁移测试表失败: %v", err)
	}
	GlobalEmbyConfig = &EmbyConfig{
		EmbyUrl:              "http://emby.local:8096",
		EmbyApiKey:           "test-key",
		EnableRefreshLibrary: 1,
	}
	IsStrmSyncTaskActiveFunc = nil
}

func TestMergeSyncPathIds(t *testing.T) {
	got := mergeSyncPathIds([]uint{3, 1, 3}, []uint{2, 1})
	want := []uint{1, 2, 3}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("合并后的 sync_path_id = %v，期望 %v", got, want)
	}
}

func TestEmbyLibraryRefreshTaskSyncPathIds(t *testing.T) {
	task := &EmbyLibraryRefreshTask{}
	task.SetSyncPathIds([]uint{5, 2})
	got := task.GetSyncPathIds()
	want := []uint{2, 5}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("解析 sync_path_ids = %v，期望 %v", got, want)
	}
	if task.SyncPathIdsStr != "[2,5]" {
		t.Fatalf("sync_path_ids 字符串 = %s，期望 [2,5]", task.SyncPathIdsStr)
	}
}

func TestRefreshTaskDefaultTimings(t *testing.T) {
	now := time.Now().Unix()
	task := newPendingEmbyLibraryRefreshTask("lib-1", "电影", []uint{1}, now)
	if task.RefreshAfterAt != now+DefaultEmbyRefreshDebounceSeconds {
		t.Fatalf("refresh_after_at = %d，期望 %d", task.RefreshAfterAt, now+DefaultEmbyRefreshDebounceSeconds)
	}
	if task.DeadlineAt != now+DefaultEmbyRefreshMaxWaitSeconds {
		t.Fatalf("deadline_at = %d，期望 %d", task.DeadlineAt, now+DefaultEmbyRefreshMaxWaitSeconds)
	}
}
