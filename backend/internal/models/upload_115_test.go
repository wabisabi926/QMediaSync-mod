package models

import (
	"context"
	"errors"
	"io"
	"log"
	"os"
	"testing"
	"time"

	"qmediasync/internal/db"
	"qmediasync/internal/helpers"
	"qmediasync/internal/v115open"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func setupUpload115ProcessedTestDB(t *testing.T) {
	t.Helper()
	if helpers.AppLogger == nil {
		helpers.AppLogger = &helpers.QLogger{Logger: log.New(io.Discard, "", 0)}
	}
	testDb, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("打开测试数据库失败: %v", err)
	}
	db.Db = testDb
	if err := db.Db.AutoMigrate(&DbUploadTask{}, &DirectoryUploadProcessedFile{}); err != nil {
		t.Fatalf("迁移测试表失败: %v", err)
	}
}

func TestBuild115RapidUploadCompleteResultFillsRemoteMtime(t *testing.T) {
	oldResolver := get115FileDetailByCid
	get115FileDetailByCid = func(_ context.Context, _ *v115open.OpenClient, fileID string) (*v115open.FileDetail, error) {
		if fileID != "file-rapid" {
			t.Fatalf("查询文件 ID = %s，期望 file-rapid", fileID)
		}
		return &v115open.FileDetail{
			FileId:       "file-rapid",
			PickCode:     "pick-detail",
			Sha1:         "sha1-detail",
			FileSizeByte: 2048,
			Ptime:        "123456",
		}, nil
	}
	t.Cleanup(func() {
		get115FileDetailByCid = oldResolver
	})

	got, err := build115RapidUploadCompleteResult(
		context.Background(),
		nil,
		&DbUploadTask{RemotePathId: "parent-1"},
		upload115LocalFileInfo{FileSha1: "sha1-local", FileSize: 1024},
		&v115open.UploadInitResult{Status: v115open.UploadInitStatusRapidUploaded, FileId: "file-rapid", PickCode: "pick-init"},
	)
	if err != nil {
		t.Fatalf("构造秒传完成结果失败: %v", err)
	}
	if got.FileId != "file-rapid" ||
		got.PickCode != "pick-detail" ||
		got.ParentId != "parent-1" ||
		got.Sha1 != "sha1-detail" ||
		got.Size != 2048 ||
		got.Mtime != 123456 {
		t.Fatalf("秒传完成结果 = %+v，期望补齐远端详情和 mtime", got)
	}
}

func TestBuild115RapidUploadCompleteResultFallsBackForDirectoryMonitor(t *testing.T) {
	oldResolver := get115FileDetailByCid
	get115FileDetailByCid = func(context.Context, *v115open.OpenClient, string) (*v115open.FileDetail, error) {
		return nil, errors.New("temporary detail error")
	}
	t.Cleanup(func() {
		get115FileDetailByCid = oldResolver
	})

	got, err := build115RapidUploadCompleteResult(
		context.Background(),
		nil,
		&DbUploadTask{Source: UploadSourceDirectoryMonitor, RemotePathId: "parent-1"},
		upload115LocalFileInfo{FileSha1: "sha1-local", FileSize: 1024},
		&v115open.UploadInitResult{Status: v115open.UploadInitStatusRapidUploaded, FileId: "file-rapid", PickCode: "pick-init"},
	)
	if err != nil {
		t.Fatalf("目录监控秒传详情查询失败时应使用 init 返回兜底: %v", err)
	}
	if got.FileId != "file-rapid" ||
		got.PickCode != "pick-init" ||
		got.ParentId != "parent-1" ||
		got.Sha1 != "sha1-local" ||
		got.Size != 1024 ||
		got.Mtime != 0 {
		t.Fatalf("兜底秒传完成结果 = %+v", got)
	}
}

func TestBuild115RapidUploadCompleteResultRequiresDetailForStrmSync(t *testing.T) {
	oldResolver := get115FileDetailByCid
	get115FileDetailByCid = func(context.Context, *v115open.OpenClient, string) (*v115open.FileDetail, error) {
		return nil, errors.New("temporary detail error")
	}
	t.Cleanup(func() {
		get115FileDetailByCid = oldResolver
	})

	_, err := build115RapidUploadCompleteResult(
		context.Background(),
		nil,
		&DbUploadTask{Source: UploadSourceStrm, RemotePathId: "parent-1"},
		upload115LocalFileInfo{FileSha1: "sha1-local", FileSize: 1024},
		&v115open.UploadInitResult{Status: v115open.UploadInitStatusRapidUploaded, FileId: "file-rapid", PickCode: "pick-init"},
	)
	if err == nil {
		t.Fatal("STRM 同步秒传详情查询失败时应返回错误，避免本地 mtime 不同步")
	}
}

func TestUpload115CompletionMarksDirectoryMonitorProcessedUploaded(t *testing.T) {
	tests := []struct {
		name   string
		result upload115TaskResult
	}{
		{
			name: "秒传完成标记 uploaded",
			result: upload115TaskResult{
				UploadResult:          UploadResultRapidUpload,
				UploadedBytes:         1024,
				CompletedRemoteFileId: "file-rapid",
				CompletedPickCode:     "pick-rapid",
			},
		},
		{
			name: "分片完成标记 uploaded",
			result: upload115TaskResult{
				UploadResult:          UploadResultMultipartUploaded,
				UploadedBytes:         2048,
				CompletedRemoteFileId: "file-multipart",
				CompletedPickCode:     "pick-multipart",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setupUpload115ProcessedTestDB(t)
			if err := db.Db.AutoMigrate(&StrmGenerationTask{}); err != nil {
				t.Fatalf("迁移 STRM 任务表失败: %v", err)
			}
			localPath := t.TempDir() + "/movie.mkv"
			if err := os.WriteFile(localPath, []byte("movie"), 0o644); err != nil {
				t.Fatalf("创建本地测试文件失败: %v", err)
			}
			info, err := os.Stat(localPath)
			if err != nil {
				t.Fatalf("读取本地测试文件失败: %v", err)
			}
			fingerprint := BuildDirectoryUploadSourceFingerprint(info.Size(), info.ModTime().UnixNano())
			task := &DbUploadTask{
				Source:            UploadSourceDirectoryMonitor,
				SourceType:        SourceType115,
				SyncPathId:        1,
				AccountId:         1,
				LocalFullPath:     localPath,
				RemoteFileId:      "/remote/movie.mkv",
				RemotePathId:      "parent-1",
				FileName:          "movie.mkv",
				SourceFingerprint: fingerprint,
				Status:            UploadStatusPending,
				Account:           &Account{BaseModel: BaseModel{ID: 1}, SourceType: SourceType115, Name: "115"},
			}
			if err := db.Db.Create(task).Error; err != nil {
				t.Fatalf("创建上传任务失败: %v", err)
			}
			record := &DirectoryUploadProcessedFile{
				SourceKey:         "source-key-uploaded",
				SourceFingerprint: task.SourceFingerprint,
				Result:            DirectoryUploadProcessedResultQueued,
				UploadTaskId:      task.ID,
				ProcessedAt:       time.Unix(100, 0).Unix(),
				LastSeenAt:        time.Unix(100, 0).Unix(),
			}
			if err := db.Db.Create(record).Error; err != nil {
				t.Fatalf("创建 processed 记录失败: %v", err)
			}
			setUpload115RunnerForTesting(t, fakeUpload115Runner{result: tt.result})

			task.Upload()

			var got DirectoryUploadProcessedFile
			if err := db.Db.Where("upload_task_id = ?", task.ID).First(&got).Error; err != nil {
				t.Fatalf("读取 processed 记录失败: %v", err)
			}
			if got.Result != DirectoryUploadProcessedResultUploaded || got.ProcessedAt <= record.ProcessedAt || got.LastSeenAt <= record.LastSeenAt {
				t.Fatalf("processed 记录 = %+v，期望 STRM 入队成功后标记 uploaded 并更新时间", got)
			}
			var strmTask StrmGenerationTask
			if err := db.Db.Where("upload_task_id = ?", task.ID).First(&strmTask).Error; err != nil {
				t.Fatalf("读取 STRM 任务失败: %v", err)
			}
			if strmTask.Status != StrmGenerationStatusPending || strmTask.FileId == "" {
				t.Fatalf("STRM 任务 = %+v，期望入队待处理", strmTask)
			}
			var gotTask DbUploadTask
			if err := db.Db.First(&gotTask, task.ID).Error; err != nil {
				t.Fatalf("读取上传任务失败: %v", err)
			}
			if gotTask.Status != UploadStatusCompleted {
				t.Fatalf("上传任务状态 = %s，期望 completed，错误：%s", gotTask.Status.String(), gotTask.Error)
			}
		})
	}
}

func TestUpload115CompletionMarksDirectoryMonitorProcessedPendingStrmBeforeEnqueue(t *testing.T) {
	tests := []struct {
		name       string
		result     UploadResult
		wantResult DirectoryUploadProcessedResult
	}{
		{name: "上传完成等待 STRM", result: UploadResultMultipartUploaded, wantResult: DirectoryUploadProcessedResultUploadedPendingStrm},
		{name: "远端已存在等待 STRM", result: UploadResultRemoteExists, wantResult: DirectoryUploadProcessedResultRemoteExistsPendingStrm},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setupUpload115ProcessedTestDB(t)
			task := &DbUploadTask{
				Source:            UploadSourceDirectoryMonitor,
				SourceFingerprint: "v1:1024:100",
				Status:            UploadStatusCompleted,
				UploadResult:      tt.result,
			}
			if err := db.Db.Create(task).Error; err != nil {
				t.Fatalf("创建上传任务失败: %v", err)
			}
			record := &DirectoryUploadProcessedFile{
				SourceKey:         "source-key-" + tt.name,
				SourceFingerprint: task.SourceFingerprint,
				Result:            DirectoryUploadProcessedResultQueued,
				UploadTaskId:      task.ID,
				ProcessedAt:       time.Unix(100, 0).Unix(),
				LastSeenAt:        time.Unix(100, 0).Unix(),
			}
			if err := db.Db.Create(record).Error; err != nil {
				t.Fatalf("创建 processed 记录失败: %v", err)
			}

			if err := task.markDirectoryUploadProcessedAfterUploadComplete(); err != nil {
				t.Fatalf("标记上传完成等待 STRM 失败: %v", err)
			}

			var got DirectoryUploadProcessedFile
			if err := db.Db.Where("upload_task_id = ?", task.ID).First(&got).Error; err != nil {
				t.Fatalf("读取 processed 记录失败: %v", err)
			}
			if got.Result != tt.wantResult {
				t.Fatalf("processed result = %s，期望 %s", got.Result, tt.wantResult)
			}
		})
	}
}

func TestUpload115SkippedAfterRapidWaitDoesNotMarkDirectoryMonitorProcessedUploaded(t *testing.T) {
	setupUpload115ProcessedTestDB(t)
	task := &DbUploadTask{
		Source:            UploadSourceDirectoryMonitor,
		SourceType:        SourceType115,
		LocalFullPath:     "/watch/movie.mkv",
		SourceFingerprint: "v1:1024:100",
		Status:            UploadStatusUploading,
	}
	if err := db.Db.Create(task).Error; err != nil {
		t.Fatalf("创建上传任务失败: %v", err)
	}
	record := &DirectoryUploadProcessedFile{
		SourceKey:         "source-key-skipped",
		SourceFingerprint: task.SourceFingerprint,
		Result:            DirectoryUploadProcessedResultQueued,
		UploadTaskId:      task.ID,
		ProcessedAt:       time.Unix(100, 0).Unix(),
		LastSeenAt:        time.Unix(100, 0).Unix(),
	}
	if err := db.Db.Create(record).Error; err != nil {
		t.Fatalf("创建 processed 记录失败: %v", err)
	}

	task.applyUpload115TaskResult(upload115TaskResult{
		UploadResult: UploadResultSkippedAfterRapidWait,
	})
	task.Complete()
	if err := task.markDirectoryUploadProcessedAfterStrm(); err != nil {
		t.Fatalf("skipped_after_rapid_wait 标记 processed 失败: %v", err)
	}

	var got DirectoryUploadProcessedFile
	if err := db.Db.Where("upload_task_id = ?", task.ID).First(&got).Error; err != nil {
		t.Fatalf("读取 processed 记录失败: %v", err)
	}
	if got.Result != DirectoryUploadProcessedResultQueued {
		t.Fatalf("processed result = %s，期望 skipped_after_rapid_wait 不标记为 uploaded", got.Result)
	}
}

func TestUpload115StrmEnqueueFailureDoesNotMarkDirectoryMonitorProcessedFailed(t *testing.T) {
	setupUpload115ProcessedTestDB(t)
	task := &DbUploadTask{
		Source:                UploadSourceDirectoryMonitor,
		SourceType:            SourceType115,
		SyncPathId:            1,
		AccountId:             1,
		LocalFullPath:         "/watch/movie.mkv",
		RemoteFileId:          "/remote/movie.mkv",
		RemotePathId:          "parent-1",
		FileName:              "movie.mkv",
		FileSize:              1024,
		SourceFingerprint:     "v1:1024:100",
		Status:                UploadStatusUploading,
		UploadResult:          UploadResultRapidUpload,
		CompletedRemoteFileId: "file-rapid",
		CompletedPickCode:     "pick-rapid",
	}
	if err := db.Db.Create(task).Error; err != nil {
		t.Fatalf("创建上传任务失败: %v", err)
	}
	record := &DirectoryUploadProcessedFile{
		SourceKey:         "source-key-strm-failed",
		SourceFingerprint: task.SourceFingerprint,
		Result:            DirectoryUploadProcessedResultUploadedPendingStrm,
		UploadTaskId:      task.ID,
		ProcessedAt:       time.Unix(100, 0).Unix(),
		LastSeenAt:        time.Unix(100, 0).Unix(),
	}
	if err := db.Db.Create(record).Error; err != nil {
		t.Fatalf("创建 processed 记录失败: %v", err)
	}

	err := task.EnqueueStrmGenerationAfterUploadAndMarkDirectoryProcessed()
	if err == nil {
		t.Fatal("STRM 表缺失时入队应失败")
	}

	var got DirectoryUploadProcessedFile
	if err := db.Db.Where("upload_task_id = ?", task.ID).First(&got).Error; err != nil {
		t.Fatalf("读取 processed 记录失败: %v", err)
	}
	if got.Result != DirectoryUploadProcessedResultStrmEnqueueFailed {
		t.Fatalf("processed result = %s，期望 STRM 入队失败标记为 strm_enqueue_failed", got.Result)
	}
}

func TestUpload115StrmEnqueueFailureMarksRemoteExistsProcessedFailed(t *testing.T) {
	setupUpload115ProcessedTestDB(t)
	localPath := t.TempDir() + "/movie.mkv"
	if err := os.WriteFile(localPath, []byte("movie"), 0o644); err != nil {
		t.Fatalf("创建本地测试文件失败: %v", err)
	}
	info, err := os.Stat(localPath)
	if err != nil {
		t.Fatalf("读取本地测试文件失败: %v", err)
	}
	task := &DbUploadTask{
		Source:            UploadSourceDirectoryMonitor,
		SourceType:        SourceType115,
		SyncPathId:        1,
		AccountId:         1,
		LocalFullPath:     localPath,
		RemoteFileId:      "/remote/movie.mkv",
		RemotePathId:      "parent-1",
		FileName:          "movie.mkv",
		FileSize:          1024,
		SourceFingerprint: BuildDirectoryUploadSourceFingerprint(info.Size(), info.ModTime().UnixNano()),
		Status:            UploadStatusPending,
		Account:           &Account{BaseModel: BaseModel{ID: 1}, SourceType: SourceType115, Name: "115"},
	}
	if err := db.Db.Create(task).Error; err != nil {
		t.Fatalf("创建上传任务失败: %v", err)
	}
	record := &DirectoryUploadProcessedFile{
		SourceKey:         "source-key-remote-exists-strm-failed",
		SourceFingerprint: task.SourceFingerprint,
		Result:            DirectoryUploadProcessedResultQueued,
		UploadTaskId:      task.ID,
		ProcessedAt:       time.Unix(100, 0).Unix(),
		LastSeenAt:        time.Unix(100, 0).Unix(),
	}
	if err := db.Db.Create(record).Error; err != nil {
		t.Fatalf("创建 processed 记录失败: %v", err)
	}
	setUpload115RunnerForTesting(t, fakeUpload115Runner{result: upload115TaskResult{
		UploadResult:          UploadResultRemoteExists,
		UploadedBytes:         1024,
		CompletedRemoteFileId: "file-remote-exists",
		CompletedPickCode:     "pick-remote-exists",
	}})

	task.Upload()

	var got DirectoryUploadProcessedFile
	if err := db.Db.Where("upload_task_id = ?", task.ID).First(&got).Error; err != nil {
		t.Fatalf("读取 processed 记录失败: %v", err)
	}
	if got.Result != DirectoryUploadProcessedResultStrmEnqueueFailed {
		t.Fatalf("processed result = %s，期望远端已存在 STRM 入队失败标记为 strm_enqueue_failed", got.Result)
	}
	var gotTask DbUploadTask
	if err := db.Db.First(&gotTask, task.ID).Error; err != nil {
		t.Fatalf("读取上传任务失败: %v", err)
	}
	if gotTask.Status != UploadStatusCompleted || gotTask.UploadResult != UploadResultRemoteExists {
		t.Fatalf("上传任务 = %+v，期望 remote_exists completed", gotTask)
	}
}

func TestUpload115MissingCompletedRemoteInfoMarksDirectoryMonitorProcessedStrmEnqueueFailed(t *testing.T) {
	setupUpload115ProcessedTestDB(t)
	task := &DbUploadTask{
		Source:            UploadSourceDirectoryMonitor,
		SourceType:        SourceType115,
		SyncPathId:        1,
		AccountId:         1,
		LocalFullPath:     "/watch/movie.mkv",
		RemoteFileId:      "/remote/movie.mkv",
		RemotePathId:      "parent-1",
		FileName:          "movie.mkv",
		FileSize:          1024,
		SourceFingerprint: "v1:1024:100",
		Status:            UploadStatusCompleted,
		UploadResult:      UploadResultRapidUpload,
	}
	if err := db.Db.Create(task).Error; err != nil {
		t.Fatalf("创建上传任务失败: %v", err)
	}
	record := &DirectoryUploadProcessedFile{
		SourceKey:         "source-key-missing-completed-remote",
		SourceFingerprint: task.SourceFingerprint,
		Result:            DirectoryUploadProcessedResultQueued,
		UploadTaskId:      task.ID,
		ProcessedAt:       time.Unix(100, 0).Unix(),
		LastSeenAt:        time.Unix(100, 0).Unix(),
	}
	if err := db.Db.Create(record).Error; err != nil {
		t.Fatalf("创建 processed 记录失败: %v", err)
	}

	if err := task.EnqueueStrmGenerationAfterUploadAndMarkDirectoryProcessed(); err == nil {
		t.Fatal("缺少远端完成信息时应返回错误")
	}

	var got DirectoryUploadProcessedFile
	if err := db.Db.Where("upload_task_id = ?", task.ID).First(&got).Error; err != nil {
		t.Fatalf("读取 processed 记录失败: %v", err)
	}
	if got.Result != DirectoryUploadProcessedResultStrmEnqueueFailed {
		t.Fatalf("processed result = %s，期望缺少远端完成信息时标记为 strm_enqueue_failed", got.Result)
	}
}

func TestUploadSkipsStaleDirectoryMonitorTaskBeforeUpload(t *testing.T) {
	setupUpload115ProcessedTestDB(t)
	filePath := t.TempDir() + "/movie.mkv"
	mtime := time.Unix(1000, 100)
	if err := os.WriteFile(filePath, []byte("movie"), 0o644); err != nil {
		t.Fatalf("创建本地测试文件失败: %v", err)
	}
	if err := os.Chtimes(filePath, mtime, mtime); err != nil {
		t.Fatalf("设置本地测试文件 mtime 失败: %v", err)
	}
	task := &DbUploadTask{
		Source:            UploadSourceDirectoryMonitor,
		SourceType:        SourceType115,
		LocalFullPath:     filePath,
		SourceFingerprint: BuildDirectoryUploadSourceFingerprint(999, 1),
		Status:            UploadStatusPending,
	}
	if err := db.Db.Create(task).Error; err != nil {
		t.Fatalf("创建上传任务失败: %v", err)
	}

	task.Upload()

	var got DbUploadTask
	if err := db.Db.First(&got, task.ID).Error; err != nil {
		t.Fatalf("读取上传任务失败: %v", err)
	}
	if got.Status != UploadStatusCancelled || got.Error == "" {
		t.Fatalf("过期目录监控任务 = %+v，期望取消并记录错误", got)
	}
}
