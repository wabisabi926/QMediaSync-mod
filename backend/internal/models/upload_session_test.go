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

func setupUploadSessionTestDB(t *testing.T) {
	t.Helper()
	if helpers.AppLogger == nil {
		helpers.AppLogger = &helpers.QLogger{Logger: log.New(io.Discard, "", 0)}
	}
	testDb, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("打开测试数据库失败: %v", err)
	}
	db.Db = testDb
	if err := db.Db.AutoMigrate(&DbUploadTask{}, &UploadSession{}); err != nil {
		t.Fatalf("迁移测试表失败: %v", err)
	}
}

func TestUploadSessionLifecycle(t *testing.T) {
	setupUploadSessionTestDB(t)

	task := &DbUploadTask{Source: UploadSourceDirectoryMonitor, SourceType: SourceType115, LocalFullPath: "/watch/movie.mkv"}
	if err := db.Db.Create(task).Error; err != nil {
		t.Fatalf("创建上传任务失败: %v", err)
	}

	session := &UploadSession{
		UploadTaskId:      task.ID,
		AccountId:         1,
		LocalFullPath:     "/watch/movie.mkv",
		FileName:          "movie.mkv",
		FileSize:          1024,
		LocalMtime:        123456,
		LocalSignature:    "1024:123456:headtail",
		FileSha1:          "sha1",
		Preid:             "preid",
		ParentFileId:      "100",
		Target:            "U_1_100",
		Status:            UploadSessionStatusInit,
		ResumeState:       UploadResumeStateNone,
		UploadedBytes:     0,
		UploadedParts:     0,
		RapidWaitUntil:    0,
		RapidWaitAttempts: 0,
	}
	if err := session.Save(); err != nil {
		t.Fatalf("保存上传会话失败: %v", err)
	}

	got, err := GetUploadSessionByUploadTaskId(task.ID)
	if err != nil {
		t.Fatalf("按上传任务查询会话失败: %v", err)
	}
	if got.UploadTaskId != task.ID || got.FileSha1 != "sha1" {
		t.Fatalf("会话查询结果 = %+v，期望 upload_task_id=%d file_sha1=sha1", got, task.ID)
	}

	got.UploadId = "oss-upload-1"
	got.UploadedBytes = 512
	if err := got.Save(); err != nil {
		t.Fatalf("更新上传进度失败: %v", err)
	}
	got, err = GetUploadSessionByUploadTaskId(task.ID)
	if err != nil {
		t.Fatalf("重新读取会话失败: %v", err)
	}
	if got.UploadId != "oss-upload-1" || got.UploadedBytes != 512 {
		t.Fatalf("更新后会话 = %+v，期望 upload_id 和 uploaded_bytes 已保存", got)
	}

	if err := got.MarkCompleted(UploadSessionCompleteResult{
		FileId:   "file-1",
		PickCode: "pick-1",
		ParentId: "100",
		Sha1:     "sha1",
		Size:     1024,
		Mtime:    123460,
	}); err != nil {
		t.Fatalf("标记上传会话完成失败: %v", err)
	}
	got, err = GetUploadSessionByUploadTaskId(task.ID)
	if err != nil {
		t.Fatalf("读取完成会话失败: %v", err)
	}
	if got.Status != UploadSessionStatusCompleted || got.CompletedFileId != "file-1" || got.CompletedPickCode != "pick-1" {
		t.Fatalf("完成后会话 = %+v，期望保存完成状态和远端定位信息", got)
	}
}

func TestUploadSessionValidateLocalFile(t *testing.T) {
	tests := []struct {
		name      string
		session   *UploadSession
		signature UploadSessionLocalSignature
		wantErr   bool
	}{
		{
			name: "本地签名完全匹配时允许恢复",
			session: &UploadSession{
				FileSize:       1024,
				LocalMtime:     100,
				FileSha1:       "sha1",
				LocalSignature: "sig",
			},
			signature: UploadSessionLocalSignature{
				FileSize:       1024,
				LocalMtime:     100,
				FileSha1:       "sha1",
				LocalSignature: "sig",
			},
		},
		{
			name: "文件大小变化时拒绝恢复",
			session: &UploadSession{
				FileSize:       1024,
				LocalMtime:     100,
				FileSha1:       "sha1",
				LocalSignature: "sig",
			},
			signature: UploadSessionLocalSignature{
				FileSize:       2048,
				LocalMtime:     100,
				FileSha1:       "sha1",
				LocalSignature: "sig",
			},
			wantErr: true,
		},
		{
			name: "SHA1 变化时拒绝恢复",
			session: &UploadSession{
				FileSize:       1024,
				LocalMtime:     100,
				FileSha1:       "sha1",
				LocalSignature: "sig",
			},
			signature: UploadSessionLocalSignature{
				FileSize:       1024,
				LocalMtime:     100,
				FileSha1:       "other",
				LocalSignature: "sig",
			},
			wantErr: true,
		},
		{
			name: "快速签名变化时拒绝恢复",
			session: &UploadSession{
				FileSize:       1024,
				LocalMtime:     100,
				FileSha1:       "sha1",
				LocalSignature: "sig",
			},
			signature: UploadSessionLocalSignature{
				FileSize:       1024,
				LocalMtime:     100,
				FileSha1:       "sha1",
				LocalSignature: "other",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.session.ValidateLocalFile(tt.signature)
			if tt.wantErr && err == nil {
				t.Fatal("期望拒绝恢复，实际返回 nil")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("期望允许恢复，实际错误: %v", err)
			}
		})
	}
}
