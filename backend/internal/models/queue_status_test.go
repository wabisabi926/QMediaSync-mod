package models

import (
	"io"
	"log"
	"testing"

	"qmediasync/internal/db"
	"qmediasync/internal/helpers"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func setupQueueStatusTestDB(t *testing.T) {
	t.Helper()
	if helpers.AppLogger == nil {
		helpers.AppLogger = &helpers.QLogger{Logger: log.New(io.Discard, "", 0)}
	}
	testDb, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("打开测试数据库失败: %v", err)
	}
	db.Db = testDb
	if err := db.Db.AutoMigrate(&DbDownloadTask{}, &DbUploadTask{}, &UploadSession{}); err != nil {
		t.Fatalf("迁移测试表失败: %v", err)
	}
	t.Cleanup(func() {
		GlobalDownloadQueue = nil
		GlobalUploadQueue = nil
	})
}

func TestDownloadQueueStatusSnapshot(t *testing.T) {
	setupQueueStatusTestDB(t)
	GlobalDownloadQueue = NewDq(1)
	GlobalDownloadQueue.running = true
	db.Db.Create(&DbDownloadTask{Status: DownloadStatusPending})
	db.Db.Create(&DbDownloadTask{Status: DownloadStatusDownloading})
	db.Db.Create(&DbDownloadTask{Status: DownloadStatusCompleted})

	snapshot := GetDownloadQueueStatusSnapshot()

	if !snapshot.Running {
		t.Fatal("下载队列快照应显示运行中")
	}
	if snapshot.Pending != 1 || snapshot.Processing != 1 || snapshot.Completed != 1 {
		t.Fatalf("下载队列状态统计 = %+v，期望 pending=1 processing=1 completed=1", snapshot)
	}
	if snapshot.Total != 3 {
		t.Fatalf("下载队列总数 = %d，期望 3", snapshot.Total)
	}
}

func TestUploadQueueStatusSnapshot(t *testing.T) {
	setupQueueStatusTestDB(t)
	GlobalUploadQueue = NewUq(1)
	GlobalUploadQueue.running = false
	db.Db.Create(&DbUploadTask{Status: UploadStatusPending})
	db.Db.Create(&DbUploadTask{Status: UploadStatusFailed})
	db.Db.Create(&DbUploadTask{Status: UploadStatusCancelled})

	snapshot := GetUploadQueueStatusSnapshot()

	if snapshot.Running {
		t.Fatal("上传队列快照应显示已停止")
	}
	if snapshot.Pending != 1 || snapshot.Failed != 1 || snapshot.Cancelled != 1 {
		t.Fatalf("上传队列状态统计 = %+v，期望 pending=1 failed=1 cancelled=1", snapshot)
	}
	if snapshot.Total != 3 {
		t.Fatalf("上传队列总数 = %d，期望 3", snapshot.Total)
	}
}

func TestUpdateUploadingToPendingPreservesUploadSession(t *testing.T) {
	setupQueueStatusTestDB(t)

	task := &DbUploadTask{
		Status:        UploadStatusUploading,
		Source:        UploadSourceDirectoryMonitor,
		SourceType:    SourceType115,
		LocalFullPath: "/watch/movie.mkv",
	}
	if err := db.Db.Create(task).Error; err != nil {
		t.Fatalf("创建上传任务失败: %v", err)
	}
	session := &UploadSession{
		UploadTaskId:  task.ID,
		Status:        UploadSessionStatusMultipart,
		ResumeState:   UploadResumeStateResumedSession,
		UploadId:      "upload-1",
		UploadedBytes: 1024,
	}
	if err := session.Save(); err != nil {
		t.Fatalf("保存上传会话失败: %v", err)
	}

	if err := UpdateUploadingToPending(); err != nil {
		t.Fatalf("恢复上传中任务失败: %v", err)
	}

	var gotTask DbUploadTask
	if err := db.Db.First(&gotTask, task.ID).Error; err != nil {
		t.Fatalf("读取上传任务失败: %v", err)
	}
	if gotTask.Status != UploadStatusPending {
		t.Fatalf("任务状态 = %s，期望 pending", gotTask.Status.String())
	}
	gotSession, err := GetUploadSessionByUploadTaskId(task.ID)
	if err != nil {
		t.Fatalf("读取上传会话失败: %v", err)
	}
	if gotSession.UploadId != "upload-1" || gotSession.UploadedBytes != 1024 {
		t.Fatalf("上传会话 = %+v，期望保留 upload_id 和进度", gotSession)
	}
}
