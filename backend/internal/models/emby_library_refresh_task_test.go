package models

import (
	"Q115-STRM/internal/db"
	"Q115-STRM/internal/helpers"
	"io"
	"log"
	"reflect"
	"testing"
	"time"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func setupEmbyRefreshTestDB(t *testing.T) {
	t.Helper()
	if helpers.AppLogger == nil {
		helpers.AppLogger = &helpers.QLogger{
			Logger: log.New(io.Discard, "", 0),
		}
	}
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

func TestRequestEmbyLibraryRefreshBySyncPathCreatesTask(t *testing.T) {
	setupEmbyRefreshTestDB(t)
	db.Db.Create(&EmbyLibrarySyncPath{LibraryId: "lib-movie", LibraryName: "电影", SyncPathId: 10})

	if err := RequestEmbyLibraryRefreshBySyncPathId(10); err != nil {
		t.Fatalf("提交刷新任务失败: %v", err)
	}

	var task EmbyLibraryRefreshTask
	if err := db.Db.Where("library_id = ?", "lib-movie").First(&task).Error; err != nil {
		t.Fatalf("查询刷新任务失败: %v", err)
	}
	if task.Status != EmbyLibraryRefreshStatusPending {
		t.Fatalf("刷新任务状态 = %s，期望 pending", task.Status)
	}
	if !reflect.DeepEqual(task.GetSyncPathIds(), []uint{10}) {
		t.Fatalf("sync_path_ids = %v，期望 [10]", task.GetSyncPathIds())
	}
}

func TestRequestEmbyLibraryRefreshMergesSameLibrary(t *testing.T) {
	setupEmbyRefreshTestDB(t)
	db.Db.Create(&EmbyLibrarySyncPath{LibraryId: "lib-movie", LibraryName: "电影", SyncPathId: 10})
	db.Db.Create(&EmbyLibrarySyncPath{LibraryId: "lib-movie", LibraryName: "电影", SyncPathId: 11})

	if err := RequestEmbyLibraryRefreshBySyncPathId(10); err != nil {
		t.Fatalf("第一次提交刷新任务失败: %v", err)
	}
	if err := RequestEmbyLibraryRefreshBySyncPathId(11); err != nil {
		t.Fatalf("第二次提交刷新任务失败: %v", err)
	}

	var tasks []EmbyLibraryRefreshTask
	db.Db.Find(&tasks)
	if len(tasks) != 1 {
		t.Fatalf("同一媒体库应合并为1条任务，实际 %d", len(tasks))
	}
	if !reflect.DeepEqual(tasks[0].GetSyncPathIds(), []uint{10, 11}) {
		t.Fatalf("sync_path_ids = %v，期望 [10 11]", tasks[0].GetSyncPathIds())
	}
}

func TestRequestEmbyLibraryRefreshSkipsDisabledConfig(t *testing.T) {
	setupEmbyRefreshTestDB(t)
	GlobalEmbyConfig.EnableRefreshLibrary = 0
	db.Db.Create(&EmbyLibrarySyncPath{LibraryId: "lib-movie", LibraryName: "电影", SyncPathId: 10})

	if err := RequestEmbyLibraryRefreshBySyncPathId(10); err != nil {
		t.Fatalf("关闭刷新时应安静跳过，实际错误: %v", err)
	}

	var total int64
	db.Db.Model(&EmbyLibraryRefreshTask{}).Count(&total)
	if total != 0 {
		t.Fatalf("关闭刷新时不应创建任务，实际 %d", total)
	}
}

func TestRequestEmbyLibraryRefreshSkipsUnlinkedSyncPath(t *testing.T) {
	setupEmbyRefreshTestDB(t)

	if err := RequestEmbyLibraryRefreshBySyncPathId(99); err != nil {
		t.Fatalf("无媒体库关联时应安静跳过，实际错误: %v", err)
	}

	var total int64
	db.Db.Model(&EmbyLibraryRefreshTask{}).Count(&total)
	if total != 0 {
		t.Fatalf("无关联时不应创建任务，实际 %d", total)
	}
}
