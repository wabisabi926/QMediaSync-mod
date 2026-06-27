package controllers

import (
	"testing"

	"qmediasync/internal/logstream"
	"qmediasync/internal/models"
)

func TestBuildSyncTaskSnapshotMessageIncludesCursorAndVersion(t *testing.T) {
	task := &models.Sync{
		BaseModel:  models.BaseModel{ID: 9, CreatedAt: 100, UpdatedAt: 120},
		SyncPathId: 5,
		Status:     models.SyncStatusInProgress,
		SubStatus:  models.SyncSubStatusProcessNetFileList,
	}
	msg := buildSyncTaskSnapshotMessage(task, []logstream.Entry{{Message: "hello"}}, 88, 3)

	if msg.Type != syncTaskStreamSnapshot {
		t.Fatalf("type = %s，期望 %s", msg.Type, syncTaskStreamSnapshot)
	}
	if msg.Version != syncTaskStreamVersion {
		t.Fatalf("version = %d，期望 %d", msg.Version, syncTaskStreamVersion)
	}
	data, ok := msg.Data.(syncTaskSnapshot)
	if !ok {
		t.Fatalf("data 类型 = %T，期望 syncTaskSnapshot", msg.Data)
	}
	if data.LogCursor != 88 {
		t.Fatalf("log_cursor = %d，期望 88", data.LogCursor)
	}
	if data.Task.ID != 9 {
		t.Fatalf("task.id = %d，期望 9", data.Task.ID)
	}
}
