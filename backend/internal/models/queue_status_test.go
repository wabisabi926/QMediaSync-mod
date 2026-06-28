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
	if err := db.Db.AutoMigrate(&DbDownloadTask{}, &DbUploadTask{}); err != nil {
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
