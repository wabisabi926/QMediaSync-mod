package syncstrm

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"qmediasync/internal/baidupan"
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
		&models.DbDownloadTask{},
		&models.DbUploadTask{},
		&models.StrmGenerationTask{},
		&models.Settings{},
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
	if err := db.Db.Create(models.SettingsGlobal).Error; err != nil {
		t.Fatalf("创建测试设置失败: %v", err)
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
			SyncDriver: &fakeDirectoryScanDriver{},
		}, nil
	}
	return service
}

func TestStrmGenerationServiceGenerateUpsertsSyncFileAndProcessesStrm(t *testing.T) {
	account, syncPath := setupStrmGenerationServiceTestDB(t)
	service := newTestGenerationService(t, syncPath, account)
	var processed *SyncFileCache
	var refreshSyncFile *models.SyncFile
	service.processStrmFile = func(_ *SyncStrm, file *SyncFileCache) error {
		processed = file
		return nil
	}
	service.compareStrm = func(_ *SyncStrm, _ *SyncFileCache) int { return 0 }
	service.requestEmbyRefreshBySyncFile = func(syncFile *models.SyncFile) error {
		refreshSyncFile = syncFile
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
	if refreshSyncFile == nil || refreshSyncFile.SyncPathId != syncPath.ID || refreshSyncFile.FileId != "file-1" {
		t.Fatalf("刷新 SyncFile = %+v，期望同步目录 %d 的 file-1", refreshSyncFile, syncPath.ID)
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
	service.requestEmbyRefreshBySyncFile = func(*models.SyncFile) error { return nil }

	if _, err := service.Generate(context.Background(), StrmGenerationInput{
		Task: &models.StrmGenerationTask{SyncPathId: syncPath.ID, AccountId: account.ID, FileId: "file-2"},
	}); err != nil {
		t.Fatalf("生成 STRM 失败: %v", err)
	}
	if processed == nil || processed.PickCode != "pick-2" || processed.Path != "/remote/show" {
		t.Fatalf("补齐后的文件 = %+v，期望包含 pick_code 和路径", processed)
	}
}

func TestStrmGenerationServiceGenerateCompletesRemoteDetailWhenMtimeMissing(t *testing.T) {
	account, syncPath := setupStrmGenerationServiceTestDB(t)
	service := newTestGenerationService(t, syncPath, account)
	var detailCalled bool
	service.detailByFileID = func(_ context.Context, _ *SyncStrm, fileID string) (*SyncFileCache, error) {
		detailCalled = true
		if fileID != "file-mtime" {
			t.Fatalf("补详情 file_id = %s，期望 file-mtime", fileID)
		}
		return &SyncFileCache{
			FileId:     "file-mtime",
			ParentId:   "parent-mtime",
			FileType:   v115open.TypeFile,
			FileName:   "movie.mkv",
			Path:       "/remote",
			FileSize:   4096,
			MTime:      345678,
			PickCode:   "pick-mtime",
			Sha1:       "sha1-mtime",
			SourceType: models.SourceType115,
		}, nil
	}
	service.compareStrm = func(_ *SyncStrm, _ *SyncFileCache) int { return 1 }
	service.processStrmFile = func(_ *SyncStrm, _ *SyncFileCache) error { return nil }
	service.requestEmbyRefreshBySyncFile = func(*models.SyncFile) error { return nil }

	if _, err := service.Generate(context.Background(), StrmGenerationInput{
		Task: &models.StrmGenerationTask{
			Source:     models.StrmGenerationSourceUploadCompleted,
			TaskType:   models.StrmGenerationTaskTypeFile,
			SyncPathId: syncPath.ID,
			AccountId:  account.ID,
			FileId:     "file-mtime",
			ParentId:   "parent-mtime",
			PickCode:   "pick-mtime",
			Path:       "/remote",
			FileName:   "movie.mkv",
			FileSize:   4096,
			Sha1:       "sha1-mtime",
		},
	}); err != nil {
		t.Fatalf("生成 STRM 失败: %v", err)
	}
	if !detailCalled {
		t.Fatal("远端时间缺失时应按 file_id 补齐远端详情")
	}

	var syncFile models.SyncFile
	if err := db.Db.Where("sync_path_id = ? AND file_id = ?", syncPath.ID, "file-mtime").First(&syncFile).Error; err != nil {
		t.Fatalf("读取 SyncFile 失败: %v", err)
	}
	if syncFile.MTime != 345678 {
		t.Fatalf("SyncFile.MTime=%d，期望补齐远端时间 345678", syncFile.MTime)
	}
}

func TestStrmGenerationServiceGenerateSkipsRefreshWhenStrmUnchanged(t *testing.T) {
	account, syncPath := setupStrmGenerationServiceTestDB(t)
	service := newTestGenerationService(t, syncPath, account)
	var refreshCalled bool
	service.processStrmFile = func(_ *SyncStrm, _ *SyncFileCache) error { return nil }
	service.compareStrm = func(_ *SyncStrm, _ *SyncFileCache) int { return 1 }
	service.requestEmbyRefreshBySyncFile = func(*models.SyncFile) error {
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

func TestStrmGenerationServiceDownloadsMatchedMetadata(t *testing.T) {
	account, syncPath := setupStrmGenerationServiceTestDB(t)
	syncPath.MetaExtArr = []string{".nfo", ".jpg"}
	service := newTestGenerationService(t, syncPath, account)
	service.compareStrm = func(_ *SyncStrm, _ *SyncFileCache) int { return 0 }
	service.processStrmFile = func(_ *SyncStrm, _ *SyncFileCache) error { return nil }
	service.requestEmbyRefreshBySyncFile = func(*models.SyncFile) error { return nil }
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
			SyncDriver: &fakeDirectoryScanDriver{
				filesByID: map[string][]*SyncFileCache{
					"parent-1": {
						{
							FileId:     "nfo-1",
							ParentId:   "parent-1",
							FileName:   "movie.nfo",
							Path:       "/remote",
							PickCode:   "pick-nfo",
							SourceType: models.SourceType115,
							FileType:   v115open.TypeFile,
						},
						{
							FileId:     "thumb-1",
							ParentId:   "parent-1",
							FileName:   "movie-thumb.jpg",
							Path:       "/remote",
							PickCode:   "pick-thumb",
							SourceType: models.SourceType115,
							FileType:   v115open.TypeFile,
						},
						{
							FileId:     "poster-1",
							ParentId:   "parent-1",
							FileName:   "poster.jpg",
							Path:       "/remote",
							PickCode:   "pick-poster",
							SourceType: models.SourceType115,
							FileType:   v115open.TypeFile,
						},
					},
				},
			},
		}, nil
	}

	_, err := service.Generate(context.Background(), StrmGenerationInput{
		Task: &models.StrmGenerationTask{
			Source:       models.StrmGenerationSourceWebhook,
			TaskType:     models.StrmGenerationTaskTypeFile,
			SyncPathId:   syncPath.ID,
			AccountId:    account.ID,
			FileId:       "file-1",
			ParentId:     "parent-1",
			PickCode:     "pick-video",
			Path:         "/remote",
			FileName:     "movie.mkv",
			DownloadMeta: true,
		},
	})
	if err != nil {
		t.Fatalf("生成 STRM 失败: %v", err)
	}
	var downloads []models.DbDownloadTask
	if err := db.Db.Order("file_name ASC").Find(&downloads).Error; err != nil {
		t.Fatalf("读取下载任务失败: %v", err)
	}
	gotNames := []string{}
	for _, task := range downloads {
		gotNames = append(gotNames, task.FileName)
	}
	wantNames := []string{"movie-thumb.jpg", "movie.nfo"}
	if !reflect.DeepEqual(gotNames, wantNames) {
		t.Fatalf("下载任务文件名 = %v，期望 %v", gotNames, wantNames)
	}
}

func TestStrmGenerationServiceIgnoresDownloadMetaForNonWebhookTask(t *testing.T) {
	account, syncPath := setupStrmGenerationServiceTestDB(t)
	syncPath.MetaExtArr = []string{".nfo"}
	service := newTestGenerationService(t, syncPath, account)
	service.compareStrm = func(_ *SyncStrm, _ *SyncFileCache) int { return 0 }
	service.processStrmFile = func(_ *SyncStrm, _ *SyncFileCache) error { return nil }
	service.requestEmbyRefreshBySyncFile = func(*models.SyncFile) error { return nil }
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
			SyncDriver: &fakeDirectoryScanDriver{
				filesByID: map[string][]*SyncFileCache{
					"parent-guard": {
						{
							FileId:     "nfo-guard",
							ParentId:   "parent-guard",
							FileName:   "movie.nfo",
							Path:       "/remote",
							PickCode:   "pick-nfo-guard",
							SourceType: models.SourceType115,
							FileType:   v115open.TypeFile,
						},
					},
				},
			},
		}, nil
	}

	result, err := service.Generate(context.Background(), StrmGenerationInput{
		Task: &models.StrmGenerationTask{
			Source:       models.StrmGenerationSourceUploadCompleted,
			TaskType:     models.StrmGenerationTaskTypeFile,
			SyncPathId:   syncPath.ID,
			AccountId:    account.ID,
			FileId:       "file-guard",
			ParentId:     "parent-guard",
			PickCode:     "pick-video-guard",
			Path:         "/remote",
			FileName:     "movie.mkv",
			DownloadMeta: true,
		},
	})
	if err != nil {
		t.Fatalf("生成 STRM 失败: %v", err)
	}
	if result.NewMeta != 0 {
		t.Fatalf("非 Webhook 任务新增元数据数 = %d，期望 0", result.NewMeta)
	}

	var count int64
	if err := db.Db.Model(&models.DbDownloadTask{}).Count(&count).Error; err != nil {
		t.Fatalf("统计下载任务失败: %v", err)
	}
	if count != 0 {
		t.Fatalf("非 Webhook 任务创建下载任务数 = %d，期望 0", count)
	}
}

func TestStrmGenerationServiceWebhookRefreshRequiresEnabledChangeOrNewMetadata(t *testing.T) {
	tests := []struct {
		name        string
		refreshEmby bool
		compare     int
	}{
		{name: "刷新开关关闭时 STRM 变更也不刷新", refreshEmby: false, compare: 0},
		{name: "刷新开关开启但 STRM 未变且无新增元数据时不刷新", refreshEmby: true, compare: 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			account, syncPath := setupStrmGenerationServiceTestDB(t)
			service := newTestGenerationService(t, syncPath, account)
			service.compareStrm = func(_ *SyncStrm, _ *SyncFileCache) int { return tt.compare }
			service.processStrmFile = func(_ *SyncStrm, _ *SyncFileCache) error { return nil }
			service.requestEmbyRefreshBySyncFile = func(*models.SyncFile) error {
				t.Fatal("Webhook file 不应走旧逐文件 Emby 刷新入口")
				return nil
			}
			service.resolveRefreshTarget = func(*models.SyncFile) (models.EmbyRefreshTarget, error) {
				t.Fatal("未满足刷新条件时不应解析 Emby 刷新目标")
				return models.EmbyRefreshTarget{}, nil
			}
			service.requestEmbyRefreshTargets = func(uint, []models.EmbyRefreshTarget) error {
				t.Fatal("未满足刷新条件时不应提交 Emby 刷新目标")
				return nil
			}

			if _, err := service.Generate(context.Background(), StrmGenerationInput{
				Task: &models.StrmGenerationTask{
					Source:      models.StrmGenerationSourceWebhook,
					TaskType:    models.StrmGenerationTaskTypeFile,
					SyncPathId:  syncPath.ID,
					AccountId:   account.ID,
					FileId:      "file-gate",
					ParentId:    "parent-gate",
					PickCode:    "pick-gate",
					Path:        "/remote",
					FileName:    "movie.mkv",
					RefreshEmby: tt.refreshEmby,
				},
			}); err != nil {
				t.Fatalf("生成 STRM 失败: %v", err)
			}
		})
	}
}

func TestStrmGenerationServiceDefaultBuilderDoesNotCreateSyncRecord(t *testing.T) {
	account, syncPath := setupStrmGenerationServiceTestDB(t)
	service := NewStrmGenerationService()
	service.compareStrm = func(_ *SyncStrm, _ *SyncFileCache) int { return 1 }
	service.processStrmFile = func(_ *SyncStrm, _ *SyncFileCache) error {
		t.Fatal("STRM 无变化时不应写入文件")
		return nil
	}
	service.requestEmbyRefreshBySyncFile = func(*models.SyncFile) error {
		t.Fatal("STRM 无变化时不应提交 Emby 刷新")
		return nil
	}

	if _, err := service.Generate(context.Background(), StrmGenerationInput{
		Task: &models.StrmGenerationTask{
			Source:     models.StrmGenerationSourceUploadCompleted,
			TaskType:   models.StrmGenerationTaskTypeFile,
			SyncPathId: syncPath.ID,
			AccountId:  account.ID,
			FileId:     "file-no-sync",
			ParentId:   "parent-no-sync",
			PickCode:   "pick-no-sync",
			Path:       "/remote",
			FileName:   "movie.mkv",
			FileSize:   1024,
			Sha1:       "sha1-no-sync",
			Mtime:      123456,
		},
	}); err != nil {
		t.Fatalf("生成 STRM 后处理失败: %v", err)
	}

	var syncCount int64
	if err := db.Db.Model(&models.Sync{}).Count(&syncCount).Error; err != nil {
		t.Fatalf("统计同步记录失败: %v", err)
	}
	if syncCount != 0 {
		t.Fatalf("同步记录数量 = %d，期望目录监控上传后处理不创建同步记录", syncCount)
	}
}

func TestStrmGenerationServiceDefaultWriterComparesStrmOnce(t *testing.T) {
	account, syncPath := setupStrmGenerationServiceTestDB(t)
	var logBuf bytes.Buffer
	helpers.AppLogger = &helpers.QLogger{Logger: log.New(&logBuf, "", 0)}

	existingStrmPath := filepath.Join(syncPath.LocalPath, "remote", "movie.strm")
	if err := os.MkdirAll(filepath.Dir(existingStrmPath), 0o755); err != nil {
		t.Fatalf("创建已有 STRM 目录失败: %v", err)
	}
	existingContent := "http://qms.local/115/url/video.mkv?pickcode=old-pick&userid=user-1"
	if err := os.WriteFile(existingStrmPath, []byte(existingContent), 0o644); err != nil {
		t.Fatalf("创建已有 STRM 失败: %v", err)
	}

	service := NewStrmGenerationService()
	service.requestEmbyRefreshBySyncFile = func(*models.SyncFile) error { return nil }

	result, err := service.Generate(context.Background(), StrmGenerationInput{
		Task: &models.StrmGenerationTask{
			Source:     models.StrmGenerationSourceUploadCompleted,
			TaskType:   models.StrmGenerationTaskTypeFile,
			SyncPathId: syncPath.ID,
			AccountId:  account.ID,
			FileId:     "file-compare-once",
			ParentId:   "parent-compare-once",
			PickCode:   "new-pick",
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
	if !result.Changed {
		t.Fatal("result.Changed = false，期望 true")
	}

	logOutput := logBuf.String()
	if count := strings.Count(logOutput, "STRM 内容 PickCode 与本地不一致"); count != 1 {
		t.Fatalf("PickCode 差异日志数量 = %d，期望 1，实际日志：%s", count, logOutput)
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
	service.requestEmbyRefreshBySyncFile = func(*models.SyncFile) error { return nil }

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
	service.requestEmbyRefreshBySyncFile = func(*models.SyncFile) error { return nil }

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
	service.requestEmbyRefreshBySyncFile = func(*models.SyncFile) error { return nil }

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

func TestProcessPendingStrmGenerationTasksDelaysParentRefreshUntilAllChildrenDone(t *testing.T) {
	account, syncPath := setupStrmGenerationServiceTestDB(t)
	parent, err := models.EnqueueStrmGenerationTask(&models.StrmGenerationTask{
		Source:      models.StrmGenerationSourceWebhook,
		TaskType:    models.StrmGenerationTaskTypeBatchFiles,
		SyncPathId:  syncPath.ID,
		AccountId:   account.ID,
		RefreshEmby: true,
		TotalItems:  2,
		Status:      models.StrmGenerationStatusCompleted,
	})
	if err != nil {
		t.Fatalf("创建父任务失败: %v", err)
	}
	for _, fileID := range []string{"file-1", "file-2"} {
		if _, err := models.EnqueueStrmGenerationTask(&models.StrmGenerationTask{
			Source:       models.StrmGenerationSourceWebhook,
			TaskType:     models.StrmGenerationTaskTypeFile,
			ParentTaskId: parent.ID,
			SyncPathId:   syncPath.ID,
			AccountId:    account.ID,
			FileId:       fileID,
			ParentId:     "parent-1",
			PickCode:     "pick-" + fileID,
			Path:         "/remote",
			FileName:     fileID + ".mkv",
			RefreshEmby:  true,
		}); err != nil {
			t.Fatalf("创建子任务失败: %v", err)
		}
	}

	service := newTestGenerationService(t, syncPath, account)
	service.compareStrm = func(_ *SyncStrm, _ *SyncFileCache) int { return 0 }
	service.processStrmFile = func(_ *SyncStrm, _ *SyncFileCache) error { return nil }
	service.requestEmbyRefreshBySyncFile = func(*models.SyncFile) error {
		t.Fatal("Webhook 父任务路径不应逐文件提交 Emby 刷新")
		return nil
	}
	service.resolveRefreshTarget = func(_ *models.SyncFile) (models.EmbyRefreshTarget, error) {
		return models.EmbyRefreshTarget{
			TargetType:        models.EmbyRefreshTargetTypeItem,
			ItemID:            "season-1",
			ItemName:          "第一季",
			ItemType:          "Season",
			Recursive:         true,
			FallbackLibraryId: "lib-tv",
		}, nil
	}
	var submitted int
	service.requestEmbyRefreshTargets = func(_ uint, targets []models.EmbyRefreshTarget) error {
		submitted++
		if len(targets) != 1 || targets[0].ItemID != "season-1" {
			t.Fatalf("提交目标 = %+v，期望单个 season-1", targets)
		}
		return nil
	}

	processed, err := ProcessPendingStrmGenerationTasks(context.Background(), service, 1)
	if err != nil {
		t.Fatalf("处理第一批失败: %v", err)
	}
	if processed != 1 || submitted != 0 {
		t.Fatalf("第一批 processed=%d submitted=%d，期望处理一个且不提交", processed, submitted)
	}
	processed, err = ProcessPendingStrmGenerationTasks(context.Background(), service, 10)
	if err != nil {
		t.Fatalf("处理第二批失败: %v", err)
	}
	if processed != 1 || submitted != 1 {
		t.Fatalf("第二批 processed=%d submitted=%d，期望最后一个子任务完成后提交一次", processed, submitted)
	}
}

func TestProcessPendingStrmGenerationTasksExpandsDirectoryScan(t *testing.T) {
	account, syncPath := setupStrmGenerationServiceTestDB(t)
	service := newTestGenerationService(t, syncPath, account)
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
			SyncDriver: &fakeDirectoryScanDriver{
				filesByID: map[string][]*SyncFileCache{
					"dir-root": {
						{
							FileId:     "file-1",
							ParentId:   "dir-root",
							FileType:   v115open.TypeFile,
							FileName:   "movie.mkv",
							Path:       "/remote/show",
							FileSize:   1024,
							MTime:      101,
							PickCode:   "pick-1",
							Sha1:       "sha1-1",
							SourceType: models.SourceType115,
						},
						{
							FileId:     "meta-1",
							ParentId:   "dir-root",
							FileType:   v115open.TypeFile,
							FileName:   "poster.jpg",
							Path:       "/remote/show",
							FileSize:   20,
							MTime:      102,
							SourceType: models.SourceType115,
						},
						{
							FileId:     "dir-season",
							ParentId:   "dir-root",
							FileType:   v115open.TypeDir,
							FileName:   "Season 01",
							Path:       "/remote/show",
							SourceType: models.SourceType115,
						},
					},
					"dir-season": {
						{
							FileId:     "file-2",
							ParentId:   "dir-season",
							FileType:   v115open.TypeFile,
							FileName:   "episode.mp4",
							Path:       "/remote/show/Season 01",
							FileSize:   2048,
							MTime:      201,
							PickCode:   "pick-2",
							Sha1:       "sha1-2",
							SourceType: models.SourceType115,
						},
					},
				},
			},
		}, nil
	}
	parent, err := models.EnqueueStrmGenerationTask(&models.StrmGenerationTask{
		Source:        models.StrmGenerationSourceWebhook,
		TaskType:      models.StrmGenerationTaskTypeDirectoryScan,
		SyncPathId:    syncPath.ID,
		AccountId:     account.ID,
		DownloadMeta:  true,
		RefreshEmby:   true,
		DirectoryPath: "/remote/show",
		RequestHash:   "webhook:directory:test",
	})
	if err != nil {
		t.Fatalf("创建目录扫描任务失败: %v", err)
	}

	processed, err := ProcessPendingStrmGenerationTasks(context.Background(), service, 10)
	if err != nil {
		t.Fatalf("处理目录扫描任务失败: %v", err)
	}
	if processed != 1 {
		t.Fatalf("处理任务数 = %d，期望 1", processed)
	}

	var gotParent models.StrmGenerationTask
	if err := db.Db.First(&gotParent, parent.ID).Error; err != nil {
		t.Fatalf("读取目录扫描父任务失败: %v", err)
	}
	if gotParent.Status != models.StrmGenerationStatusCompleted || gotParent.TotalItems != 2 || gotParent.AcceptedItems != 0 || gotParent.FailedItems != 0 {
		t.Fatalf("目录扫描父任务 = %+v，期望 completed 且 total_items=2", gotParent)
	}

	var children []models.StrmGenerationTask
	if err := db.Db.Where("parent_task_id = ?", parent.ID).Order("file_id ASC").Find(&children).Error; err != nil {
		t.Fatalf("读取目录扫描子任务失败: %v", err)
	}
	if len(children) != 2 {
		t.Fatalf("子任务数量 = %d，期望 2: %+v", len(children), children)
	}
	if children[0].FileId != "file-1" || children[0].Path != "/remote/show" || children[0].FileName != "movie.mkv" {
		t.Fatalf("第一个子任务 = %+v，期望 movie.mkv", children[0])
	}
	if children[1].FileId != "file-2" || children[1].Path != "/remote/show/Season 01" || children[1].FileName != "episode.mp4" {
		t.Fatalf("第二个子任务 = %+v，期望 episode.mp4", children[1])
	}
	for _, child := range children {
		if child.DownloadMeta != gotParent.DownloadMeta || child.RefreshEmby != gotParent.RefreshEmby {
			t.Fatalf("目录扫描子任务未继承父任务开关: child=%+v parent=%+v", child, gotParent)
		}
	}
}

func TestProcessPendingStrmGenerationTasksExpandsDirectoryScanByDirectoryID(t *testing.T) {
	account, syncPath := setupStrmGenerationServiceTestDB(t)
	service := newTestGenerationService(t, syncPath, account)
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
			SyncDriver: &fakeDirectoryScanDriver{
				filesByID: map[string][]*SyncFileCache{
					"dir-root": {
						{
							FileId:     "file-1",
							ParentId:   "dir-root",
							FileType:   v115open.TypeFile,
							FileName:   "movie.mkv",
							Path:       "/remote/show",
							FileSize:   1024,
							MTime:      101,
							PickCode:   "pick-1",
							Sha1:       "sha1-1",
							SourceType: models.SourceType115,
						},
					},
				},
				detailsByID: map[string]*SyncFileCache{
					"dir-root": {
						FileId:     "dir-root",
						ParentId:   "root",
						FileType:   v115open.TypeDir,
						FileName:   "show",
						Path:       "/remote",
						SourceType: models.SourceType115,
					},
				},
			},
		}, nil
	}
	parent, err := models.EnqueueStrmGenerationTask(&models.StrmGenerationTask{
		Source:      models.StrmGenerationSourceWebhook,
		TaskType:    models.StrmGenerationTaskTypeDirectoryScan,
		SyncPathId:  syncPath.ID,
		AccountId:   account.ID,
		DirectoryId: "dir-root",
		RequestHash: "webhook:directory:id-only",
	})
	if err != nil {
		t.Fatalf("创建目录扫描任务失败: %v", err)
	}

	processed, err := ProcessPendingStrmGenerationTasks(context.Background(), service, 10)
	if err != nil {
		t.Fatalf("处理目录扫描任务失败: %v", err)
	}
	if processed != 1 {
		t.Fatalf("处理任务数 = %d，期望 1", processed)
	}

	var gotParent models.StrmGenerationTask
	if err := db.Db.First(&gotParent, parent.ID).Error; err != nil {
		t.Fatalf("读取目录扫描父任务失败: %v", err)
	}
	if gotParent.Status != models.StrmGenerationStatusCompleted || gotParent.TotalItems != 1 {
		t.Fatalf("目录扫描父任务 = %+v，期望 completed 且 total_items=1", gotParent)
	}

	var child models.StrmGenerationTask
	if err := db.Db.Where("parent_task_id = ?", parent.ID).First(&child).Error; err != nil {
		t.Fatalf("读取目录扫描子任务失败: %v", err)
	}
	if child.FileId != "file-1" || child.Path != "/remote/show" || child.FileName != "movie.mkv" {
		t.Fatalf("子任务 = %+v，期望 movie.mkv", child)
	}
}

func TestProcessPendingStrmGenerationTasksCallsSourceCleanupAfterCompletedUploadTask(t *testing.T) {
	account, syncPath := setupStrmGenerationServiceTestDB(t)
	service := newTestGenerationService(t, syncPath, account)
	service.compareStrm = func(_ *SyncStrm, _ *SyncFileCache) int { return 1 }
	service.processStrmFile = func(_ *SyncStrm, _ *SyncFileCache) error { return nil }
	service.requestEmbyRefreshBySyncFile = func(*models.SyncFile) error { return nil }

	var cleanupUploadTaskIDs []uint
	oldCleanup := cleanupSourceAfterStrmSuccess
	cleanupSourceAfterStrmSuccess = func(uploadTaskID uint) error {
		cleanupUploadTaskIDs = append(cleanupUploadTaskIDs, uploadTaskID)
		return nil
	}
	t.Cleanup(func() {
		cleanupSourceAfterStrmSuccess = oldCleanup
	})

	if _, err := models.EnqueueStrmGenerationTask(&models.StrmGenerationTask{
		Source:       models.StrmGenerationSourceUploadCompleted,
		TaskType:     models.StrmGenerationTaskTypeFile,
		UploadTaskId: 42,
		SyncPathId:   syncPath.ID,
		AccountId:    account.ID,
		FileId:       "file-cleanup",
		ParentId:     "parent-cleanup",
		PickCode:     "pick-cleanup",
		Path:         "/remote",
		FileName:     "cleanup.mkv",
	}); err != nil {
		t.Fatalf("创建清理任务失败: %v", err)
	}
	if _, err := ProcessPendingStrmGenerationTasks(context.Background(), service, 10); err != nil {
		t.Fatalf("处理 STRM 任务失败: %v", err)
	}
	if len(cleanupUploadTaskIDs) != 1 || cleanupUploadTaskIDs[0] != 42 {
		t.Fatalf("cleanup ids=%v，期望只清理上传任务 42", cleanupUploadTaskIDs)
	}
}

func TestProcessPendingStrmGenerationTasksCopiesDirectoryUploadMetadataBeforeCleanup(t *testing.T) {
	account, syncPath := setupStrmGenerationServiceTestDB(t)
	service := newTestGenerationService(t, syncPath, account)
	service.compareStrm = func(_ *SyncStrm, _ *SyncFileCache) int {
		t.Fatal("元数据后处理不应比较 STRM 内容")
		return 1
	}
	service.processStrmFile = func(_ *SyncStrm, _ *SyncFileCache) error {
		t.Fatal("元数据后处理不应写入 STRM 文件")
		return nil
	}
	service.requestEmbyRefreshBySyncFile = func(*models.SyncFile) error { return nil }

	sourceDir := t.TempDir()
	sourcePath := filepath.Join(sourceDir, "show", "Season 01", "episode.nfo")
	if err := os.MkdirAll(filepath.Dir(sourcePath), 0o777); err != nil {
		t.Fatalf("创建源文件目录失败: %v", err)
	}
	content := []byte("metadata")
	if err := os.WriteFile(sourcePath, content, 0o666); err != nil {
		t.Fatalf("写入源文件失败: %v", err)
	}

	uploadTask := &models.DbUploadTask{
		Source:                models.UploadSourceDirectoryMonitor,
		AccountId:             account.ID,
		SyncPathId:            syncPath.ID,
		SourceType:            models.SourceType115,
		LocalFullPath:         sourcePath,
		RelativePath:          "show/Season 01/episode.nfo",
		RemoteFileId:          "/remote/show/Season 01/episode.nfo",
		RemotePathId:          "parent-meta",
		FileName:              "episode.nfo",
		Status:                models.UploadStatusCompleted,
		FileSize:              int64(len(content)),
		UploadResult:          models.UploadResultMultipartUploaded,
		CompletedRemoteFileId: "file-meta",
		CompletedPickCode:     "pick-meta",
		SourceCleanupStatus:   models.UploadSourceCleanupStatusPending,
	}
	if err := db.Db.Create(uploadTask).Error; err != nil {
		t.Fatalf("创建目录监控上传任务失败: %v", err)
	}

	targetFile := (&SyncFileCache{
		FileId:     "file-meta",
		ParentId:   "parent-meta",
		FileType:   v115open.TypeFile,
		FileName:   "episode.nfo",
		Path:       "/remote/show/Season 01",
		PickCode:   "pick-meta",
		SourceType: models.SourceType115,
		IsMeta:     true,
	}).GetLocalFilePath(syncPath.LocalPath, syncPath.RemotePath)

	cleanupCalled := false
	oldCleanup := cleanupSourceAfterStrmSuccess
	cleanupSourceAfterStrmSuccess = func(uploadTaskID uint) error {
		cleanupCalled = true
		if uploadTaskID != uploadTask.ID {
			t.Fatalf("清理 upload_task_id = %d，期望 %d", uploadTaskID, uploadTask.ID)
		}
		got, err := os.ReadFile(targetFile)
		if err != nil {
			t.Fatalf("源文件清理前应已复制元数据到 STRM 路径: %v", err)
		}
		if string(got) != string(content) {
			t.Fatalf("复制后的元数据内容 = %q，期望 %q", got, content)
		}
		if _, err := os.Stat(sourcePath); err != nil {
			t.Fatalf("源文件清理前源文件应仍存在: %v", err)
		}
		return nil
	}
	t.Cleanup(func() {
		cleanupSourceAfterStrmSuccess = oldCleanup
	})

	if _, err := models.EnqueueStrmGenerationTask(&models.StrmGenerationTask{
		Source:       models.StrmGenerationSourceUploadCompleted,
		TaskType:     models.StrmGenerationTaskTypeFile,
		UploadTaskId: uploadTask.ID,
		SyncPathId:   syncPath.ID,
		AccountId:    account.ID,
		FileId:       "file-meta",
		ParentId:     "parent-meta",
		PickCode:     "pick-meta",
		Path:         "/remote/show/Season 01",
		FileName:     "episode.nfo",
		FileSize:     int64(len(content)),
	}); err != nil {
		t.Fatalf("创建元数据 STRM 后处理任务失败: %v", err)
	}
	if _, err := ProcessPendingStrmGenerationTasks(context.Background(), service, 10); err != nil {
		t.Fatalf("处理 STRM 任务失败: %v", err)
	}
	if !cleanupCalled {
		t.Fatal("目录监控元数据任务完成后应触发源文件清理")
	}

	var syncFile models.SyncFile
	if err := db.Db.Where("sync_path_id = ? AND file_id = ?", syncPath.ID, "file-meta").First(&syncFile).Error; err != nil {
		t.Fatalf("读取元数据 SyncFile 失败: %v", err)
	}
	if !syncFile.IsMeta || syncFile.IsVideo || syncFile.LocalFilePath != targetFile {
		t.Fatalf("元数据 SyncFile = %+v，期望记录 STRM 路径 %s", syncFile, targetFile)
	}
}

func TestProcessPendingStrmGenerationTasksSkipsCleanupWhenFailedOrNoUploadTask(t *testing.T) {
	account, syncPath := setupStrmGenerationServiceTestDB(t)
	service := newTestGenerationService(t, syncPath, account)
	service.compareStrm = func(_ *SyncStrm, file *SyncFileCache) int {
		if file.FileId == "file-fail-cleanup" {
			return 0
		}
		return 1
	}
	service.processStrmFile = func(_ *SyncStrm, file *SyncFileCache) error {
		if file.FileId == "file-fail-cleanup" {
			return errors.New("写入失败")
		}
		return nil
	}
	service.requestEmbyRefreshBySyncFile = func(*models.SyncFile) error { return nil }

	cleanupCalled := false
	oldCleanup := cleanupSourceAfterStrmSuccess
	cleanupSourceAfterStrmSuccess = func(uploadTaskID uint) error {
		cleanupCalled = true
		return nil
	}
	t.Cleanup(func() {
		cleanupSourceAfterStrmSuccess = oldCleanup
	})

	if _, err := models.EnqueueStrmGenerationTask(&models.StrmGenerationTask{
		Source:     models.StrmGenerationSourceUploadCompleted,
		TaskType:   models.StrmGenerationTaskTypeFile,
		SyncPathId: syncPath.ID,
		AccountId:  account.ID,
		FileId:     "file-no-upload",
		ParentId:   "parent-no-upload",
		PickCode:   "pick-no-upload",
		Path:       "/remote",
		FileName:   "no-upload.mkv",
	}); err != nil {
		t.Fatalf("创建无上传 ID 任务失败: %v", err)
	}
	if _, err := models.EnqueueStrmGenerationTask(&models.StrmGenerationTask{
		Source:       models.StrmGenerationSourceUploadCompleted,
		TaskType:     models.StrmGenerationTaskTypeFile,
		UploadTaskId: 43,
		SyncPathId:   syncPath.ID,
		AccountId:    account.ID,
		FileId:       "file-fail-cleanup",
		ParentId:     "parent-fail-cleanup",
		PickCode:     "pick-fail-cleanup",
		Path:         "/remote",
		FileName:     "fail-cleanup.mkv",
	}); err != nil {
		t.Fatalf("创建失败任务失败: %v", err)
	}
	if _, err := ProcessPendingStrmGenerationTasks(context.Background(), service, 10); err != nil {
		t.Fatalf("处理 STRM 任务失败: %v", err)
	}
	if cleanupCalled {
		t.Fatal("STRM 失败或无 upload_task_id 时不应触发源文件清理")
	}
}

func TestStrmRemotePathWithinRoot(t *testing.T) {
	tests := []struct {
		name       string
		remotePath string
		basePath   string
		want       bool
	}{
		{name: "根目录包含子路径", remotePath: "/remote/show", basePath: "/", want: true},
		{name: "同级前缀不误判", remotePath: "/remote2/show", basePath: "/remote", want: false},
		{name: "同步目录包含自身", remotePath: "/remote", basePath: "/remote", want: true},
		{name: "同步目录包含子路径", remotePath: "/remote/show", basePath: "/remote", want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := strmRemotePathWithin(tt.remotePath, tt.basePath); got != tt.want {
				t.Fatalf("strmRemotePathWithin(%q, %q) = %v，期望 %v", tt.remotePath, tt.basePath, got, tt.want)
			}
		})
	}
}

type fakeDirectoryScanDriver struct {
	filesByID   map[string][]*SyncFileCache
	detailsByID map[string]*SyncFileCache
}

func (driver *fakeDirectoryScanDriver) GetNetFileFiles(_ context.Context, _ string, parentPathID string) ([]*SyncFileCache, error) {
	return driver.filesByID[parentPathID], nil
}

func (driver *fakeDirectoryScanDriver) GetPathIdByPath(_ context.Context, path string) (string, error) {
	switch path {
	case "/remote/show":
		return "dir-root", nil
	case "/remote/show/Season 01":
		return "dir-season", nil
	default:
		return "", nil
	}
}

func (driver *fakeDirectoryScanDriver) SetSyncStrm(*SyncStrm) {}

func (driver *fakeDirectoryScanDriver) MakeStrmContent(*SyncFileCache) string { return "" }

func (driver *fakeDirectoryScanDriver) CreateDirRecursively(context.Context, string) (string, string, error) {
	return "", "", nil
}

func (driver *fakeDirectoryScanDriver) GetTotalFileCount(context.Context) (int64, string, error) {
	return 0, "", nil
}

func (driver *fakeDirectoryScanDriver) GetDirsByPathId(context.Context, string) ([]pathQueueItem, error) {
	return nil, nil
}

func (driver *fakeDirectoryScanDriver) GetFilesByPathId(context.Context, string, int, int) ([]v115open.File, error) {
	return nil, nil
}

func (driver *fakeDirectoryScanDriver) GetFilesByPathMtime(context.Context, string, int, int, int64) (*baidupan.FileListAllResponse, error) {
	return nil, nil
}

func (driver *fakeDirectoryScanDriver) DetailByFileId(_ context.Context, fileID string) (*SyncFileCache, error) {
	return driver.detailsByID[fileID], nil
}

func (driver *fakeDirectoryScanDriver) DeleteFile(context.Context, string, []string) error {
	return nil
}
