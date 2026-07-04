package syncstrm

import (
	"context"
	"errors"
	"io"
	"log"
	"os"
	"path/filepath"
	"testing"

	"qmediasync/internal/db"
	"qmediasync/internal/helpers"
	"qmediasync/internal/models"
	"qmediasync/internal/v115open"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func setupStrmGenerationServiceTestDB(t *testing.T) (*models.Account, *models.SyncPath) {
	t.Helper()
	helpers.AppLogger = &helpers.QLogger{Logger: log.New(io.Discard, "", 0)}
	helpers.V115Log = &helpers.QLogger{Logger: log.New(io.Discard, "", 0)}
	testDB, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("打开测试数据库失败: %v", err)
	}
	db.Db = testDB
	if err := db.Db.AutoMigrate(
		&models.Account{},
		&models.SyncPath{},
		&models.Sync{},
		&models.SyncFile{},
		&models.StrmGenerationTask{},
		&models.EmbyConfig{},
		&models.EmbyLibrary{},
		&models.EmbyLibrarySyncPath{},
		&models.EmbyLibraryRefreshTask{},
	); err != nil {
		t.Fatalf("迁移测试表失败: %v", err)
	}
	models.SettingsGlobal = &models.Settings{
		SettingThreads: models.SettingThreads{FileDetailThreads: 2},
		SettingStrm: models.SettingStrm{
			VideoExtArr: []string{".mkv", ".mp4"},
			MetaExtArr:  []string{".nfo"},
			AddPath:     3,
		},
	}
	account := &models.Account{
		SourceType: models.SourceType115,
		Name:       "115",
		UserId:     "user-1",
	}
	if err := db.Db.Create(account).Error; err != nil {
		t.Fatalf("创建账号失败: %v", err)
	}
	syncPath := &models.SyncPath{
		AccountId:    account.ID,
		SourceType:   models.SourceType115,
		LocalPath:    t.TempDir(),
		RemotePath:   "/remote",
		BaseCid:      "root",
		CustomConfig: true,
		SettingStrm: models.SettingStrm{
			StrmBaseUrl: "http://qms.local",
			VideoExtArr: []string{".mkv", ".mp4"},
			MetaExtArr:  []string{".nfo"},
			AddPath:     3,
		},
	}
	if err := db.Db.Create(syncPath).Error; err != nil {
		t.Fatalf("创建同步目录失败: %v", err)
	}
	return account, syncPath
}

func newTestGenerationService(t *testing.T, syncPath *models.SyncPath, account *models.Account) *StrmGenerationService {
	t.Helper()
	service := NewStrmGenerationService()
	service.buildSyncer = func(_ *models.SyncPath, _ *models.Account) (*SyncStrm, error) {
		return &SyncStrm{
			Account:      account,
			SyncPathId:   syncPath.ID,
			SourcePath:   syncPath.RemotePath,
			SourcePathId: syncPath.BaseCid,
			TargetPath:   syncPath.LocalPath,
			Config: SyncStrmConfig{
				StrmBaseUrl:     syncPath.StrmBaseUrl,
				StrmUrlNeedPath: syncPath.AddPath,
				VideoExt:        syncPath.VideoExtArr,
				MetaExt:         syncPath.MetaExtArr,
			},
		}, nil
	}
	return service
}

func TestStrmGenerationServiceGenerateUpsertsSyncFileAndProcessesStrm(t *testing.T) {
	account, syncPath := setupStrmGenerationServiceTestDB(t)
	service := newTestGenerationService(t, syncPath, account)
	var processed *SyncFileCache
	var refreshSyncPathID uint
	service.processStrmFile = func(_ *SyncStrm, file *SyncFileCache) error {
		processed = file
		return nil
	}
	service.compareStrm = func(_ *SyncStrm, _ *SyncFileCache) int { return 0 }
	service.requestEmbyRefresh = func(syncPathID uint) error {
		refreshSyncPathID = syncPathID
		return nil
	}

	result, err := service.Generate(context.Background(), StrmGenerationInput{
		Task: &models.StrmGenerationTask{
			Source:     models.StrmGenerationSourceUploadCompleted,
			TaskType:   models.StrmGenerationTaskTypeFile,
			SyncPathId: syncPath.ID,
			AccountId:  account.ID,
			FileId:     "file-1",
			ParentId:   "parent-1",
			PickCode:   "pick-1",
			Path:       "/remote",
			FileName:   "movie.mkv",
			FileSize:   1024,
			Sha1:       "sha1",
			Mtime:      123456,
		},
	})
	if err != nil {
		t.Fatalf("生成 STRM 失败: %v", err)
	}
	if processed == nil || processed.FileId != "file-1" || !processed.IsVideo {
		t.Fatalf("处理的文件 = %+v，期望 file-1 视频文件", processed)
	}
	if !result.Changed {
		t.Fatal("result.Changed = false，期望 true")
	}
	if refreshSyncPathID != syncPath.ID {
		t.Fatalf("刷新同步目录 ID = %d，期望 %d", refreshSyncPathID, syncPath.ID)
	}

	var syncFile models.SyncFile
	if err := db.Db.Where("sync_path_id = ? AND file_id = ?", syncPath.ID, "file-1").First(&syncFile).Error; err != nil {
		t.Fatalf("读取 SyncFile 失败: %v", err)
	}
	if syncFile.PickCode != "pick-1" || syncFile.LocalFilePath == "" {
		t.Fatalf("SyncFile = %+v，期望写入远端定位和本地 STRM 路径", syncFile)
	}
}

func TestStrmGenerationServiceGenerateCompletesDetailByFileID(t *testing.T) {
	account, syncPath := setupStrmGenerationServiceTestDB(t)
	service := newTestGenerationService(t, syncPath, account)
	service.detailByFileID = func(_ context.Context, _ *SyncStrm, fileID string) (*SyncFileCache, error) {
		if fileID != "file-2" {
			t.Fatalf("补详情 file_id = %s，期望 file-2", fileID)
		}
		return &SyncFileCache{
			FileId:     "file-2",
			ParentId:   "parent-2",
			FileType:   v115open.TypeFile,
			FileName:   "episode.mkv",
			Path:       "/remote/show",
			FileSize:   2048,
			MTime:      234567,
			PickCode:   "pick-2",
			Sha1:       "sha1-2",
			SourceType: models.SourceType115,
		}, nil
	}
	var processed *SyncFileCache
	service.processStrmFile = func(_ *SyncStrm, file *SyncFileCache) error {
		processed = file
		return nil
	}
	service.compareStrm = func(_ *SyncStrm, _ *SyncFileCache) int { return 0 }
	service.requestEmbyRefresh = func(uint) error { return nil }

	if _, err := service.Generate(context.Background(), StrmGenerationInput{
		Task: &models.StrmGenerationTask{SyncPathId: syncPath.ID, AccountId: account.ID, FileId: "file-2"},
	}); err != nil {
		t.Fatalf("生成 STRM 失败: %v", err)
	}
	if processed == nil || processed.PickCode != "pick-2" || processed.Path != "/remote/show" {
		t.Fatalf("补齐后的文件 = %+v，期望包含 pick_code 和路径", processed)
	}
}

func TestStrmGenerationServiceGenerateSkipsRefreshWhenStrmUnchanged(t *testing.T) {
	account, syncPath := setupStrmGenerationServiceTestDB(t)
	service := newTestGenerationService(t, syncPath, account)
	var refreshCalled bool
	service.processStrmFile = func(_ *SyncStrm, _ *SyncFileCache) error { return nil }
	service.compareStrm = func(_ *SyncStrm, _ *SyncFileCache) int { return 1 }
	service.requestEmbyRefresh = func(uint) error {
		refreshCalled = true
		return nil
	}

	result, err := service.Generate(context.Background(), StrmGenerationInput{
		Task: &models.StrmGenerationTask{
			SyncPathId: syncPath.ID,
			AccountId:  account.ID,
			FileId:     "file-3",
			ParentId:   "parent-3",
			PickCode:   "pick-3",
			Path:       "/remote",
			FileName:   "movie.mkv",
		},
	})
	if err != nil {
		t.Fatalf("生成 STRM 失败: %v", err)
	}
	if result.Changed {
		t.Fatal("result.Changed = true，期望 false")
	}
	if refreshCalled {
		t.Fatal("STRM 无变化时不应提交 Emby 刷新")
	}
}

func TestStrmGenerationServiceCleansOldStrmAfterRemotePathChanges(t *testing.T) {
	account, syncPath := setupStrmGenerationServiceTestDB(t)
	oldStrmPath := filepath.Join(syncPath.LocalPath, "old", "movie.strm")
	if err := os.MkdirAll(filepath.Dir(oldStrmPath), 0o755); err != nil {
		t.Fatalf("创建旧 STRM 目录失败: %v", err)
	}
	if err := os.WriteFile(oldStrmPath, []byte("old"), 0o644); err != nil {
		t.Fatalf("创建旧 STRM 失败: %v", err)
	}
	if err := db.Db.Create(&models.SyncFile{
		AccountId:     account.ID,
		SyncPathId:    syncPath.ID,
		SourceType:    models.SourceType115,
		FileId:        "file-move",
		ParentId:      "old-parent",
		FileName:      "movie.mkv",
		Path:          "/old",
		PickCode:      "pick-move",
		LocalFilePath: filepath.ToSlash(oldStrmPath),
		IsVideo:       true,
	}).Error; err != nil {
		t.Fatalf("创建旧 SyncFile 失败: %v", err)
	}

	newStrmPath := filepath.Join(syncPath.LocalPath, "remote", "movie.strm")
	service := newTestGenerationService(t, syncPath, account)
	service.processStrmFile = func(_ *SyncStrm, _ *SyncFileCache) error {
		if err := os.MkdirAll(filepath.Dir(newStrmPath), 0o755); err != nil {
			return err
		}
		return os.WriteFile(newStrmPath, []byte("new"), 0o644)
	}
	service.compareStrm = func(_ *SyncStrm, _ *SyncFileCache) int { return 0 }
	service.requestEmbyRefresh = func(uint) error { return nil }

	result, err := service.Generate(context.Background(), StrmGenerationInput{
		Task: &models.StrmGenerationTask{
			SyncPathId: syncPath.ID,
			AccountId:  account.ID,
			FileId:     "file-move",
			ParentId:   "new-parent",
			PickCode:   "pick-move",
			Path:       "/remote",
			FileName:   "movie.mkv",
		},
	})
	if err != nil {
		t.Fatalf("生成 STRM 失败: %v", err)
	}
	if !result.Changed {
		t.Fatal("result.Changed = false，期望 true")
	}
	if _, err := os.Stat(oldStrmPath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("旧 STRM 状态错误: %v，期望已删除", err)
	}
	if _, err := os.Stat(newStrmPath); err != nil {
		t.Fatalf("新 STRM 应保留: %v", err)
	}
}

func TestStrmGenerationServiceKeepsNewStrmWhenOldPathCleanupFails(t *testing.T) {
	account, syncPath := setupStrmGenerationServiceTestDB(t)
	oldDir := filepath.Join(syncPath.LocalPath, "old.strm")
	if err := os.MkdirAll(oldDir, 0o755); err != nil {
		t.Fatalf("创建旧路径目录失败: %v", err)
	}
	if err := os.WriteFile(filepath.Join(oldDir, "child"), []byte("x"), 0o644); err != nil {
		t.Fatalf("创建旧路径占用文件失败: %v", err)
	}
	if err := db.Db.Create(&models.SyncFile{
		AccountId:     account.ID,
		SyncPathId:    syncPath.ID,
		SourceType:    models.SourceType115,
		FileId:        "file-4",
		ParentId:      "old-parent",
		FileName:      "old.mkv",
		Path:          "/old",
		PickCode:      "pick-4",
		LocalFilePath: oldDir,
		IsVideo:       true,
	}).Error; err != nil {
		t.Fatalf("创建旧 SyncFile 失败: %v", err)
	}

	newStrmPath := filepath.Join(syncPath.LocalPath, "remote", "new.strm")
	service := newTestGenerationService(t, syncPath, account)
	service.processStrmFile = func(_ *SyncStrm, _ *SyncFileCache) error {
		if err := os.MkdirAll(filepath.Dir(newStrmPath), 0o755); err != nil {
			return err
		}
		return os.WriteFile(newStrmPath, []byte("new"), 0o644)
	}
	service.compareStrm = func(_ *SyncStrm, _ *SyncFileCache) int { return 0 }
	service.requestEmbyRefresh = func(uint) error { return nil }

	_, err := service.Generate(context.Background(), StrmGenerationInput{
		Task: &models.StrmGenerationTask{
			SyncPathId: syncPath.ID,
			AccountId:  account.ID,
			FileId:     "file-4",
			ParentId:   "new-parent",
			PickCode:   "pick-4",
			Path:       "/remote",
			FileName:   "new.mkv",
		},
	})
	if err == nil {
		t.Fatal("旧路径清理失败时应返回错误，保留任务可重试")
	}
	if !errors.Is(err, errOldStrmCleanupFailed) {
		t.Fatalf("错误 = %v，期望包含 errOldStrmCleanupFailed", err)
	}
	if _, statErr := os.Stat(newStrmPath); statErr != nil {
		t.Fatalf("新 STRM 不应因旧路径清理失败被删除: %v", statErr)
	}
}

func TestProcessPendingStrmGenerationTasksMarksCompletedAndFailed(t *testing.T) {
	account, syncPath := setupStrmGenerationServiceTestDB(t)
	service := newTestGenerationService(t, syncPath, account)
	service.compareStrm = func(_ *SyncStrm, file *SyncFileCache) int {
		if file.FileId == "file-fail" {
			return 0
		}
		return 1
	}
	service.processStrmFile = func(_ *SyncStrm, file *SyncFileCache) error {
		if file.FileId == "file-fail" {
			return errors.New("写入失败")
		}
		return nil
	}
	service.requestEmbyRefresh = func(uint) error { return nil }

	completed, err := models.EnqueueStrmGenerationTask(&models.StrmGenerationTask{
		Source:     models.StrmGenerationSourceUploadCompleted,
		TaskType:   models.StrmGenerationTaskTypeFile,
		SyncPathId: syncPath.ID,
		AccountId:  account.ID,
		FileId:     "file-ok",
		ParentId:   "parent-ok",
		PickCode:   "pick-ok",
		Path:       "/remote",
		FileName:   "ok.mkv",
	})
	if err != nil {
		t.Fatalf("创建成功任务失败: %v", err)
	}
	failed, err := models.EnqueueStrmGenerationTask(&models.StrmGenerationTask{
		Source:     models.StrmGenerationSourceUploadCompleted,
		TaskType:   models.StrmGenerationTaskTypeFile,
		SyncPathId: syncPath.ID,
		AccountId:  account.ID,
		FileId:     "file-fail",
		ParentId:   "parent-fail",
		PickCode:   "pick-fail",
		Path:       "/remote",
		FileName:   "fail.mkv",
	})
	if err != nil {
		t.Fatalf("创建失败任务失败: %v", err)
	}

	processed, err := ProcessPendingStrmGenerationTasks(context.Background(), service, 10)
	if err != nil {
		t.Fatalf("处理待生成 STRM 任务失败: %v", err)
	}
	if processed != 2 {
		t.Fatalf("处理任务数 = %d，期望 2", processed)
	}

	var completedTask models.StrmGenerationTask
	if err := db.Db.First(&completedTask, completed.ID).Error; err != nil {
		t.Fatalf("读取完成任务失败: %v", err)
	}
	if completedTask.Status != models.StrmGenerationStatusCompleted || completedTask.LastError != "" {
		t.Fatalf("完成任务状态 = %+v，期望 completed 且无错误", completedTask)
	}
	var failedTask models.StrmGenerationTask
	if err := db.Db.First(&failedTask, failed.ID).Error; err != nil {
		t.Fatalf("读取失败任务失败: %v", err)
	}
	if failedTask.Status != models.StrmGenerationStatusFailed || failedTask.RetryCount != 1 || failedTask.LastError == "" {
		t.Fatalf("失败任务状态 = %+v，期望 failed/retry_count=1/last_error", failedTask)
	}
}
