package models

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"qmediasync/internal/db"
	"qmediasync/internal/v115open"
)

func TestCanRetryFailedDownloadTask(t *testing.T) {
	task := &DbDownloadTask{Status: DownloadStatusFailed, Error: "network", RetryCount: 0}
	task.PrepareDownloadRetry(1)
	if task.Status != DownloadStatusPending {
		t.Fatal("下载失败任务应回到等待中")
	}
	if task.Error != "" {
		t.Fatal("重试时应清空错误信息")
	}
	if task.RetryCount != 1 {
		t.Fatal("重试次数应递增")
	}
}

func TestDownloadTaskDoesNotRetryBeyondLimit(t *testing.T) {
	task := &DbDownloadTask{Status: DownloadStatusFailed, RetryCount: 1}
	if task.CanRetry(1) {
		t.Fatal("达到最大重试次数后不应继续重试")
	}
}

func TestCanRetryFailedUploadTask(t *testing.T) {
	task := &DbUploadTask{Status: UploadStatusFailed, Error: "network", RetryCount: 0}
	task.PrepareUploadRetry(1)
	if task.Status != UploadStatusPending {
		t.Fatal("上传失败任务应回到等待中")
	}
	if task.Error != "" {
		t.Fatal("重试时应清空错误信息")
	}
	if task.RetryCount != 1 {
		t.Fatal("重试次数应递增")
	}
}

func TestUploadTaskDoesNotRetryBeyondLimit(t *testing.T) {
	task := &DbUploadTask{Status: UploadStatusFailed, RetryCount: 1}
	if task.CanRetry(1) {
		t.Fatal("达到最大重试次数后不应继续重试")
	}
}

func TestPrepare115UploadSessionResumesValidSession(t *testing.T) {
	setupQueueStatusTestDB(t)

	task := &DbUploadTask{
		AccountId:      1,
		Status:         UploadStatusFailed,
		Source:         UploadSourceDirectoryMonitor,
		SourceType:     SourceType115,
		LocalFullPath:  "/watch/movie.mkv",
		FileName:       "movie.mkv",
		FileSize:       1024,
		RemotePathId:   "100",
		UploadedBytes:  512,
		UploadResult:   UploadResultUnknown,
		ResumeState:    UploadResumeStateNone,
		RapidWaitUntil: 0,
	}
	if err := db.Db.Create(task).Error; err != nil {
		t.Fatalf("创建上传任务失败: %v", err)
	}
	session := &UploadSession{
		UploadTaskId:   task.ID,
		AccountId:      task.AccountId,
		LocalFullPath:  task.LocalFullPath,
		FileName:       task.FileName,
		FileSize:       task.FileSize,
		LocalMtime:     100,
		LocalSignature: "sig",
		FileSha1:       "sha1",
		Preid:          "preid",
		ParentFileId:   task.RemotePathId,
		PickCode:       "pick-1",
		Bucket:         "bucket-1",
		Object:         "object-1",
		UploadId:       "upload-1",
		PartSize:       256,
		UploadedBytes:  512,
		UploadedParts:  2,
		Status:         UploadSessionStatusMultipart,
		ResumeState:    UploadResumeStateNewSession,
	}
	if err := session.Save(); err != nil {
		t.Fatalf("保存上传会话失败: %v", err)
	}

	got, err := task.prepare115UploadSession(upload115LocalFileInfo{
		FileName:       task.FileName,
		FileSize:       task.FileSize,
		LocalMtime:     100,
		LocalSignature: "sig",
		FileSha1:       "sha1",
		Preid:          "preid",
	})
	if err != nil {
		t.Fatalf("准备上传会话失败: %v", err)
	}

	if got.ID != session.ID {
		t.Fatalf("会话 ID = %d，期望复用已有会话 %d", got.ID, session.ID)
	}
	if got.ResumeState != UploadResumeStateResumedSession || got.LastResumeAt == 0 {
		t.Fatalf("会话恢复状态 = %+v，期望 resumed_session 并记录恢复时间", got)
	}
	if task.ResumeState != UploadResumeStateResumedSession || task.UploadedBytes != 512 {
		t.Fatalf("任务续传字段 = %+v，期望同步 session 进度", task)
	}
}

func TestPrepare115UploadSessionAbortsChangedSignature(t *testing.T) {
	setupQueueStatusTestDB(t)

	task := &DbUploadTask{
		AccountId:     1,
		Status:        UploadStatusPending,
		Source:        UploadSourceDirectoryMonitor,
		SourceType:    SourceType115,
		LocalFullPath: "/watch/movie.mkv",
		FileName:      "movie.mkv",
		FileSize:      1024,
		RemotePathId:  "100",
	}
	if err := db.Db.Create(task).Error; err != nil {
		t.Fatalf("创建上传任务失败: %v", err)
	}
	session := &UploadSession{
		UploadTaskId:   task.ID,
		AccountId:      task.AccountId,
		LocalFullPath:  task.LocalFullPath,
		FileName:       task.FileName,
		FileSize:       task.FileSize,
		LocalMtime:     100,
		LocalSignature: "old-sig",
		FileSha1:       "old-sha1",
		Preid:          "old-preid",
		ParentFileId:   task.RemotePathId,
		PickCode:       "pick-1",
		UploadId:       "upload-1",
		Status:         UploadSessionStatusMultipart,
		ResumeState:    UploadResumeStateResumedSession,
	}
	if err := session.Save(); err != nil {
		t.Fatalf("保存上传会话失败: %v", err)
	}

	_, err := task.prepare115UploadSession(upload115LocalFileInfo{
		FileName:       task.FileName,
		FileSize:       task.FileSize,
		LocalMtime:     time.Now().Unix(),
		LocalSignature: "new-sig",
		FileSha1:       "new-sha1",
		Preid:          "new-preid",
	})
	if err == nil {
		t.Fatal("本地文件签名变化时应拒绝复用旧 session")
	}

	got, readErr := GetUploadSessionByUploadTaskId(task.ID)
	if readErr != nil {
		t.Fatalf("读取上传会话失败: %v", readErr)
	}
	if got.Status != UploadSessionStatusAborted || got.LastError == "" {
		t.Fatalf("会话状态 = %+v，期望 aborted 并记录错误", got)
	}
}

func TestUpload115FilePersistsResultAndEnqueuesStrmTask(t *testing.T) {
	setupQueueStatusTestDB(t)
	if err := db.Db.AutoMigrate(&StrmGenerationTask{}); err != nil {
		t.Fatalf("迁移 STRM 生成任务表失败: %v", err)
	}

	localPath := filepath.Join(t.TempDir(), "movie.mkv")
	if err := os.WriteFile(localPath, []byte("movie-content"), 0o644); err != nil {
		t.Fatalf("创建本地测试文件失败: %v", err)
	}
	task := &DbUploadTask{
		AccountId:     1,
		SyncPathId:    10,
		Status:        UploadStatusPending,
		Source:        UploadSourceDirectoryMonitor,
		SourceType:    SourceType115,
		LocalFullPath: localPath,
		FileName:      "movie.mkv",
		FileSize:      13,
		RemoteFileId:  "/remote/movie.mkv",
		RemotePathId:  "100",
		Account:       &Account{BaseModel: BaseModel{ID: 1}, SourceType: SourceType115, Name: "115"},
	}
	if err := db.Db.Create(task).Error; err != nil {
		t.Fatalf("创建上传任务失败: %v", err)
	}

	setUpload115RunnerForTesting(t, fakeUpload115Runner{
		result: upload115TaskResult{
			UploadResult:          UploadResultMultipartUploaded,
			ResumeState:           UploadResumeStateResumedSession,
			UploadedBytes:         13,
			TotalParts:            2,
			UploadedParts:         2,
			CompletedRemoteFileId: "file-1",
			CompletedPickCode:     "pick-1",
			CompletedParentId:     "100",
			CompletedSha1:         "sha1",
			CompletedSize:         13,
			CompletedMtime:        123456,
		},
	})

	task.Upload()

	var gotTask DbUploadTask
	if err := db.Db.First(&gotTask, task.ID).Error; err != nil {
		t.Fatalf("读取上传任务失败: %v", err)
	}
	if gotTask.Status != UploadStatusCompleted {
		t.Fatalf("任务状态 = %s，期望 completed，错误：%s", gotTask.Status.String(), gotTask.Error)
	}
	if gotTask.UploadResult != UploadResultMultipartUploaded ||
		gotTask.ResumeState != UploadResumeStateResumedSession ||
		gotTask.UploadedBytes != 13 ||
		gotTask.CompletedRemoteFileId != "file-1" ||
		gotTask.CompletedPickCode != "pick-1" {
		t.Fatalf("上传结果字段 = %+v，期望保存完成结果和续传状态", gotTask)
	}

	var strmTask StrmGenerationTask
	if err := db.Db.Where("upload_task_id = ?", task.ID).First(&strmTask).Error; err != nil {
		t.Fatalf("上传成功后应创建 STRM 生成任务: %v", err)
	}
	if strmTask.Source != StrmGenerationSourceUploadCompleted ||
		strmTask.SyncPathId != task.SyncPathId ||
		strmTask.FileId != "file-1" ||
		strmTask.PickCode != "pick-1" {
		t.Fatalf("STRM 生成任务 = %+v，期望写入上传完成的远端定位信息", strmTask)
	}
}

type fakeUpload115Runner struct {
	result upload115TaskResult
	err    error
}

func (runner fakeUpload115Runner) Upload(_ context.Context, _ *DbUploadTask, _ *v115open.OpenClient) (upload115TaskResult, error) {
	return runner.result, runner.err
}
