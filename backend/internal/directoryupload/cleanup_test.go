package directoryupload

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"qmediasync/internal/db"
	"qmediasync/internal/models"
)

func TestCleanupSourceAfterStrmSuccessDeletesFileAndEmptyParents(t *testing.T) {
	setupDirectoryUploadServiceTestDB(t)
	monitorPath := t.TempDir()
	nested := filepath.Join(monitorPath, "show", "Season 01")
	if err := ensureDir(nested); err != nil {
		t.Fatalf("创建测试目录失败: %v", err)
	}
	filePath := filepath.Join(nested, "episode.mkv")
	writeFileWithMtime(t, filePath, []byte("episode"), time.Now())
	_, rule := createDirectoryUploadRuleForTest(t, monitorPath)
	rule.DeleteSourceAfterSuccess = true
	if err := db.Db.Save(rule).Error; err != nil {
		t.Fatalf("保存目录上传规则失败: %v", err)
	}
	task := createCleanupUploadTask(t, rule, filePath, models.UploadResultMultipartUploaded)
	createCleanupStrmTask(t, task.ID, models.StrmGenerationStatusCompleted)

	if err := CleanupSourceAfterStrmSuccess(task.ID); err != nil {
		t.Fatalf("源文件清理失败: %v", err)
	}
	assertPathMissing(t, filePath)
	assertPathMissing(t, nested)
	assertPathMissing(t, filepath.Join(monitorPath, "show"))
	assertPathExists(t, monitorPath)

	var got models.DbUploadTask
	if err := db.Db.First(&got, task.ID).Error; err != nil {
		t.Fatalf("读取上传任务失败: %v", err)
	}
	if got.SourceCleanupStatus != models.UploadSourceCleanupStatusCompleted || got.SourceDeletedAt == 0 || got.SourceCleanupError != "" {
		t.Fatalf("清理状态 = %+v，期望 completed", got)
	}
}

func TestCleanupSourceAfterStrmSuccessKeepsSourceWhenDisabledOrStrmFailed(t *testing.T) {
	tests := []struct {
		name       string
		deleteFlag bool
		strmStatus models.StrmGenerationStatus
	}{
		{name: "删除开关关闭", deleteFlag: false, strmStatus: models.StrmGenerationStatusCompleted},
		{name: "STRM 未完成", deleteFlag: true, strmStatus: models.StrmGenerationStatusFailed},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setupDirectoryUploadServiceTestDB(t)
			monitorPath := t.TempDir()
			filePath := filepath.Join(monitorPath, "movie.mkv")
			writeFileWithMtime(t, filePath, []byte("movie"), time.Now())
			_, rule := createDirectoryUploadRuleForTest(t, monitorPath)
			rule.DeleteSourceAfterSuccess = tt.deleteFlag
			if err := db.Db.Save(rule).Error; err != nil {
				t.Fatalf("保存目录上传规则失败: %v", err)
			}
			task := createCleanupUploadTask(t, rule, filePath, models.UploadResultMultipartUploaded)
			createCleanupStrmTask(t, task.ID, tt.strmStatus)

			if err := CleanupSourceAfterStrmSuccess(task.ID); err != nil {
				t.Fatalf("源文件清理返回错误: %v", err)
			}
			assertPathExists(t, filePath)
		})
	}
}

func TestCleanupSourceAfterStrmSuccessRequiresSafeUploadResult(t *testing.T) {
	setupDirectoryUploadServiceTestDB(t)
	monitorPath := t.TempDir()
	filePath := filepath.Join(monitorPath, "movie.mkv")
	writeFileWithMtime(t, filePath, []byte("movie"), time.Now())
	_, rule := createDirectoryUploadRuleForTest(t, monitorPath)
	rule.DeleteSourceAfterSuccess = true
	if err := db.Db.Save(rule).Error; err != nil {
		t.Fatalf("保存目录上传规则失败: %v", err)
	}
	task := createCleanupUploadTask(t, rule, filePath, models.UploadResultSkippedAfterRapidWait)
	createCleanupStrmTask(t, task.ID, models.StrmGenerationStatusCompleted)

	if err := CleanupSourceAfterStrmSuccess(task.ID); err != nil {
		t.Fatalf("源文件清理返回错误: %v", err)
	}
	assertPathExists(t, filePath)
}

func TestCleanupSourceAfterStrmSuccessKeepsReplacedSourceFile(t *testing.T) {
	setupDirectoryUploadServiceTestDB(t)
	monitorPath := t.TempDir()
	filePath := filepath.Join(monitorPath, "movie.mkv")
	originalMtime := time.Unix(1000, 0)
	writeFileWithMtime(t, filePath, []byte("movie"), originalMtime)
	_, rule := createDirectoryUploadRuleForTest(t, monitorPath)
	rule.DeleteSourceAfterSuccess = true
	if err := db.Db.Save(rule).Error; err != nil {
		t.Fatalf("保存目录上传规则失败: %v", err)
	}
	task := createCleanupUploadTask(t, rule, filePath, models.UploadResultMultipartUploaded)
	createCleanupUploadSession(t, task.ID, filePath)
	createCleanupStrmTask(t, task.ID, models.StrmGenerationStatusCompleted)
	replacedMtime := originalMtime.Add(time.Hour)
	writeFileWithMtime(t, filePath, []byte("new movie content"), replacedMtime)

	if err := CleanupSourceAfterStrmSuccess(task.ID); err == nil {
		t.Fatal("源文件已被替换时应返回错误")
	}
	assertPathExists(t, filePath)

	var got models.DbUploadTask
	if err := db.Db.First(&got, task.ID).Error; err != nil {
		t.Fatalf("读取上传任务失败: %v", err)
	}
	if got.SourceCleanupStatus != models.UploadSourceCleanupStatusFailed || got.SourceCleanupError == "" {
		t.Fatalf("清理状态 = %+v，期望 failed 并记录错误", got)
	}
}

func TestCleanupSourceAfterStrmSuccessKeepsSourceWhenUploadFailed(t *testing.T) {
	setupDirectoryUploadServiceTestDB(t)
	monitorPath := t.TempDir()
	filePath := filepath.Join(monitorPath, "movie.mkv")
	writeFileWithMtime(t, filePath, []byte("movie"), time.Now())
	_, rule := createDirectoryUploadRuleForTest(t, monitorPath)
	rule.DeleteSourceAfterSuccess = true
	if err := db.Db.Save(rule).Error; err != nil {
		t.Fatalf("保存目录上传规则失败: %v", err)
	}
	task := createCleanupUploadTask(t, rule, filePath, models.UploadResultMultipartUploaded)
	task.Status = models.UploadStatusFailed
	if err := db.Db.Save(task).Error; err != nil {
		t.Fatalf("保存上传失败状态失败: %v", err)
	}
	createCleanupStrmTask(t, task.ID, models.StrmGenerationStatusCompleted)

	if err := CleanupSourceAfterStrmSuccess(task.ID); err != nil {
		t.Fatalf("源文件清理返回错误: %v", err)
	}
	assertPathExists(t, filePath)
}

func TestCleanupSourceAfterStrmSuccessStopsAtNonEmptyParent(t *testing.T) {
	setupDirectoryUploadServiceTestDB(t)
	monitorPath := t.TempDir()
	nested := filepath.Join(monitorPath, "show", "Season 01")
	if err := ensureDir(nested); err != nil {
		t.Fatalf("创建测试目录失败: %v", err)
	}
	filePath := filepath.Join(nested, "episode.mkv")
	siblingPath := filepath.Join(monitorPath, "show", "poster.jpg")
	writeFileWithMtime(t, filePath, []byte("episode"), time.Now())
	writeFileWithMtime(t, siblingPath, []byte("poster"), time.Now())
	_, rule := createDirectoryUploadRuleForTest(t, monitorPath)
	rule.DeleteSourceAfterSuccess = true
	if err := db.Db.Save(rule).Error; err != nil {
		t.Fatalf("保存目录上传规则失败: %v", err)
	}
	task := createCleanupUploadTask(t, rule, filePath, models.UploadResultRapidUpload)
	createCleanupStrmTask(t, task.ID, models.StrmGenerationStatusCompleted)

	if err := CleanupSourceAfterStrmSuccess(task.ID); err != nil {
		t.Fatalf("源文件清理失败: %v", err)
	}
	assertPathMissing(t, filePath)
	assertPathMissing(t, nested)
	assertPathExists(t, filepath.Join(monitorPath, "show"))
	assertPathExists(t, siblingPath)
}

func TestCleanupSourceAfterStrmSuccessRecordsPathBoundaryError(t *testing.T) {
	setupDirectoryUploadServiceTestDB(t)
	monitorPath := t.TempDir()
	outsidePath := filepath.Join(t.TempDir(), "movie.mkv")
	writeFileWithMtime(t, outsidePath, []byte("movie"), time.Now())
	_, rule := createDirectoryUploadRuleForTest(t, monitorPath)
	rule.DeleteSourceAfterSuccess = true
	if err := db.Db.Save(rule).Error; err != nil {
		t.Fatalf("保存目录上传规则失败: %v", err)
	}
	task := createCleanupUploadTask(t, rule, outsidePath, models.UploadResultMultipartUploaded)
	createCleanupStrmTask(t, task.ID, models.StrmGenerationStatusCompleted)

	if err := CleanupSourceAfterStrmSuccess(task.ID); err == nil {
		t.Fatal("源文件越界时应返回错误")
	}
	assertPathExists(t, outsidePath)

	var got models.DbUploadTask
	if err := db.Db.First(&got, task.ID).Error; err != nil {
		t.Fatalf("读取上传任务失败: %v", err)
	}
	if got.SourceCleanupStatus != models.UploadSourceCleanupStatusFailed || got.SourceCleanupError == "" {
		t.Fatalf("清理状态 = %+v，期望 failed 并记录错误", got)
	}
}

func createCleanupUploadTask(t *testing.T, rule *models.DirectoryUploadRule, filePath string, result models.UploadResult) *models.DbUploadTask {
	t.Helper()
	info, err := os.Stat(filePath)
	if err != nil {
		t.Fatalf("读取测试文件信息失败: %v", err)
	}
	task := &models.DbUploadTask{
		Source:                models.UploadSourceDirectoryMonitor,
		AccountId:             rule.AccountId,
		SyncPathId:            rule.SyncPathId,
		SourceType:            models.SourceType115,
		LocalFullPath:         filePath,
		RelativePath:          mustRel(t, rule.MonitorPath, filePath),
		RemoteFileId:          "/remote/" + filepath.Base(filePath),
		RemotePathId:          rule.RemoteRootId,
		FileName:              filepath.Base(filePath),
		Status:                models.UploadStatusCompleted,
		FileSize:              info.Size(),
		LocalMtime:            info.ModTime().Unix(),
		UploadResult:          result,
		CompletedRemoteFileId: "remote-file",
		CompletedPickCode:     "pick-code",
		SourceCleanupStatus:   models.UploadSourceCleanupStatusPending,
	}
	if err := db.Db.Create(task).Error; err != nil {
		t.Fatalf("创建上传任务失败: %v", err)
	}
	return task
}

func createCleanupUploadSession(t *testing.T, uploadTaskID uint, filePath string) {
	t.Helper()
	info, err := os.Stat(filePath)
	if err != nil {
		t.Fatalf("读取测试文件信息失败: %v", err)
	}
	if err := db.Db.Create(&models.UploadSession{
		UploadTaskId:  uploadTaskID,
		LocalFullPath: filePath,
		FileSize:      info.Size(),
		LocalMtime:    info.ModTime().Unix(),
		Status:        models.UploadSessionStatusCompleted,
	}).Error; err != nil {
		t.Fatalf("创建上传会话失败: %v", err)
	}
}

func createCleanupStrmTask(t *testing.T, uploadTaskID uint, status models.StrmGenerationStatus) {
	t.Helper()
	if err := db.Db.Create(&models.StrmGenerationTask{
		Source:       models.StrmGenerationSourceUploadCompleted,
		TaskType:     models.StrmGenerationTaskTypeFile,
		UploadTaskId: uploadTaskID,
		SyncPathId:   1,
		Status:       status,
	}).Error; err != nil {
		t.Fatalf("创建 STRM 任务失败: %v", err)
	}
}
