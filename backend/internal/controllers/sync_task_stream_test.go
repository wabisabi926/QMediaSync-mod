package controllers

import (
	"os"
	"regexp"
	"strings"
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

func TestSyncTaskStreamSubscribesBeforeSnapshotReload(t *testing.T) {
	source, err := os.ReadFile("sync_task_stream.go")
	if err != nil {
		t.Fatalf("读取 sync_task_stream.go 失败：%v", err)
	}

	content := string(source)
	subscribeIndex := strings.Index(content, "taskEvents, unsubscribeTask := ws.GlobalSyncTaskHub.Subscribe")
	reloadIndex := strings.Index(content, "latestTask, err := models.GetSyncByID(idReq.ID)")
	if subscribeIndex < 0 {
		t.Fatal("未找到同步任务事件订阅代码")
	}
	if reloadIndex < 0 {
		t.Fatal("未找到订阅后的任务快照重读代码")
	}
	if subscribeIndex > reloadIndex {
		t.Fatal("同步任务 stream 应先订阅事件，再重读 snapshot，避免订阅前状态变化丢失")
	}
}

func TestSyncTaskStreamSendsDeletedCompleteForMissingTask(t *testing.T) {
	source, err := os.ReadFile("sync_task_stream.go")
	if err != nil {
		t.Fatalf("读取 sync_task_stream.go 失败：%v", err)
	}

	content := string(source)
	if strings.Contains(content, "http.StatusNotFound") {
		t.Fatal("同步任务 stream 不应在记录不存在时返回 HTTP 404，应发送 deleted complete 消息")
	}
	if !strings.Contains(content, "writeMissingSyncTaskComplete(c, idReq.ID)") {
		t.Fatal("记录不存在时未进入 deleted complete helper")
	}
	if !strings.Contains(content, "SyncID:    syncID") || !regexp.MustCompile(`Deleted:\s+true`).MatchString(content) {
		t.Fatal("未找到记录不存在时发送 deleted complete 消息的代码")
	}
}
