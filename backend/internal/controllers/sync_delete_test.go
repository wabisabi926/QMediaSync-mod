package controllers

import (
	"io"
	"log"
	"testing"

	"qmediasync/internal/db"
	"qmediasync/internal/helpers"
	"qmediasync/internal/models"
)

func setupSyncDeleteControllerTestDB(t *testing.T) {
	t.Helper()

	originalConfigDir := helpers.ConfigDir
	originalGlobalConfig := helpers.GlobalConfig
	originalLogger := helpers.AppLogger
	t.Cleanup(func() {
		helpers.ConfigDir = originalConfigDir
		helpers.GlobalConfig = originalGlobalConfig
		helpers.AppLogger = originalLogger
	})

	setupControllerTestDB(t, &models.Sync{})
	helpers.ConfigDir = t.TempDir()
	helpers.GlobalConfig = *helpers.MakeDefaultConfig()
	helpers.AppLogger = &helpers.QLogger{Logger: log.New(io.Discard, "", 0)}
}

func TestDeleteSyncRecordsByIDs汇总删除结果(t *testing.T) {
	setupSyncDeleteControllerTestDB(t)
	completed := &models.Sync{SyncPathId: 1, Status: models.SyncStatusCompleted, LocalPath: "/local", RemotePath: "/remote"}
	running := &models.Sync{SyncPathId: 1, Status: models.SyncStatusInProgress, LocalPath: "/local", RemotePath: "/remote"}
	if err := db.Db.Create(completed).Error; err != nil {
		t.Fatalf("创建已完成同步记录失败: %v", err)
	}
	if err := db.Db.Create(running).Error; err != nil {
		t.Fatalf("创建进行中同步记录失败: %v", err)
	}

	result := deleteSyncRecordsByIDs([]uint{completed.ID, running.ID, 404})

	if len(result.DeletedIDs) != 1 || result.DeletedIDs[0] != completed.ID {
		t.Fatalf("DeletedIDs = %+v，期望只包含 %d", result.DeletedIDs, completed.ID)
	}
	if len(result.Failures) != 2 {
		t.Fatalf("Failures = %+v，期望 2 条失败", result.Failures)
	}
	if result.Failures[0].ID != running.ID || result.Failures[0].Reason == "" {
		t.Fatalf("第一条失败 = %+v，期望进行中记录失败且有原因", result.Failures[0])
	}
	if result.Failures[1].ID != 404 || result.Failures[1].Reason == "" {
		t.Fatalf("第二条失败 = %+v，期望缺失记录失败且有原因", result.Failures[1])
	}
}
