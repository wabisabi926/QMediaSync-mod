package directoryupload

import (
	"context"
	"io"
	"log"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"qmediasync/internal/db"
	"qmediasync/internal/helpers"
	"qmediasync/internal/models"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

type fakeRemoteClient struct {
	parentID string
	files    map[string]*RemoteFile
}

func (c *fakeRemoteClient) EnsureDir(context.Context, *models.DirectoryUploadRule, string) (RemoteDirectory, error) {
	return RemoteDirectory{ID: c.parentID, Path: "/remote/movie"}, nil
}

func (c *fakeRemoteClient) FindFile(_ context.Context, _ string, fileName string) (*RemoteFile, error) {
	if c.files == nil {
		return nil, nil
	}
	return c.files[fileName], nil
}

func setupDirectoryUploadServiceTestDB(t *testing.T) {
	t.Helper()
	if helpers.AppLogger == nil {
		helpers.AppLogger = &helpers.QLogger{Logger: log.New(io.Discard, "", 0)}
	}
	testDB, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("打开测试数据库失败: %v", err)
	}
	db.Db = testDB
	if err := db.Db.AutoMigrate(
		&models.Account{},
		&models.SyncPath{},
		&models.DirectoryUploadRule{},
		&models.DbUploadTask{},
		&models.UploadSession{},
		&models.StrmGenerationTask{},
	); err != nil {
		t.Fatalf("迁移测试表失败: %v", err)
	}
	models.SettingsGlobal = &models.Settings{
		SettingStrm: models.SettingStrm{
			VideoExtArr:  []string{".mkv", ".mp4"},
			MetaExtArr:   []string{".nfo"},
			MinVideoSize: 0,
		},
	}
}

func createDirectoryUploadRuleForTest(t *testing.T, monitorPath string) (*models.SyncPath, *models.DirectoryUploadRule) {
	t.Helper()
	syncPath := &models.SyncPath{
		SourceType:  models.SourceType115,
		AccountId:   1,
		BaseCid:     "root",
		LocalPath:   filepath.Join(t.TempDir(), "strm"),
		RemotePath:  "/remote",
		SettingStrm: models.SettingStrm{VideoExtArr: []string{".mkv", ".mp4"}, MinVideoSize: 0},
	}
	if err := db.Db.Create(syncPath).Error; err != nil {
		t.Fatalf("创建同步目录失败: %v", err)
	}
	rule := &models.DirectoryUploadRule{
		SyncPathId:                    syncPath.ID,
		AccountId:                     1,
		Enabled:                       true,
		MonitorPath:                   monitorPath,
		RemoteRootPath:                "/remote",
		RemoteRootId:                  "remote-root",
		Recursive:                     true,
		WatchMode:                     models.DirectoryUploadWatchModeAuto,
		StabilitySeconds:              0,
		StabilityCheckIntervalSeconds: 1,
		StabilityRequiredCount:        1,
		ProcessedCacheTTLSeconds:      600,
	}
	if err := db.Db.Create(rule).Error; err != nil {
		t.Fatalf("创建目录监控规则失败: %v", err)
	}
	return syncPath, rule
}

func TestScanRuleAddsRecursiveVideoFilesToStabilityQueue(t *testing.T) {
	setupDirectoryUploadServiceTestDB(t)
	monitorPath := t.TempDir()
	nested := filepath.Join(monitorPath, "show", "Season 01")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatalf("创建测试目录失败: %v", err)
	}
	writeFileWithMtime(t, filepath.Join(monitorPath, "movie.mkv"), []byte("movie"), time.Now())
	writeFileWithMtime(t, filepath.Join(nested, "episode.mp4"), []byte("episode"), time.Now())
	writeFileWithMtime(t, filepath.Join(nested, "ignore.tmp"), []byte("tmp"), time.Now())
	writeFileWithMtime(t, filepath.Join(nested, ".hidden.mkv"), []byte("hidden"), time.Now())

	_, rule := createDirectoryUploadRuleForTest(t, monitorPath)
	rule.StabilitySeconds = 0
	rule.StabilityRequiredCount = 1
	service := NewService(ServiceOptions{})
	accepted, err := service.ScanRule(context.Background(), rule)
	if err != nil {
		t.Fatalf("扫描目录失败: %v", err)
	}
	if accepted != 2 {
		t.Fatalf("accepted=%d，期望 2 个视频文件进入稳定性队列", accepted)
	}

	got := service.PendingPaths(rule.ID)
	want := []string{
		filepath.Join(monitorPath, "movie.mkv"),
		filepath.Join(nested, "episode.mp4"),
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("pending paths=%v，期望 %v", got, want)
	}
}

func TestScanRuleStableFilesCreateUploadTasks(t *testing.T) {
	setupDirectoryUploadServiceTestDB(t)
	monitorPath := t.TempDir()
	nested := filepath.Join(monitorPath, "show", "Season 01")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatalf("创建测试目录失败: %v", err)
	}
	writeFileWithMtime(t, filepath.Join(monitorPath, "movie.mkv"), []byte("movie"), time.Now())
	writeFileWithMtime(t, filepath.Join(nested, "episode.mp4"), []byte("episode"), time.Now())
	writeFileWithMtime(t, filepath.Join(nested, "ignore.tmp"), []byte("tmp"), time.Now())

	_, rule := createDirectoryUploadRuleForTest(t, monitorPath)
	rule.StabilitySeconds = 0
	rule.StabilityRequiredCount = 1
	service := NewService(ServiceOptions{})
	service.SetRemoteClient(&fakeRemoteClient{parentID: "remote-root"})
	if accepted, err := service.ScanRule(context.Background(), rule); err != nil || accepted != 2 {
		t.Fatalf("扫描目录 accepted=%d err=%v，期望 2 个候选视频", accepted, err)
	}
	if ready, err := service.CheckStableFiles(rule); err != nil || len(ready) != 0 {
		t.Fatalf("首次稳定性检查 ready=%v err=%v，期望未就绪", ready, err)
	}
	ready, err := service.CheckStableFiles(rule)
	if err != nil {
		t.Fatalf("第二次稳定性检查失败: %v", err)
	}
	if len(ready) != 2 {
		t.Fatalf("稳定文件数量 = %d，期望 2: %+v", len(ready), ready)
	}
	for _, file := range ready {
		if err := service.HandleStableFile(context.Background(), rule, file.Path); err != nil {
			t.Fatalf("创建上传任务失败: %v", err)
		}
	}

	var tasks []models.DbUploadTask
	if err := db.Db.Order("id ASC").Find(&tasks).Error; err != nil {
		t.Fatalf("读取上传任务失败: %v", err)
	}
	if len(tasks) != 2 {
		t.Fatalf("上传任务数量 = %d，期望 2", len(tasks))
	}
	for _, task := range tasks {
		if task.Source != models.UploadSourceDirectoryMonitor || task.Status != models.UploadStatusPending {
			t.Fatalf("上传任务 = %+v，期望目录监控 pending 任务", task)
		}
	}
}

func TestServiceDedupesProcessedFileUntilTTLExpires(t *testing.T) {
	setupDirectoryUploadServiceTestDB(t)
	clock := &fakeClock{now: time.Unix(300, 0)}
	monitorPath := t.TempDir()
	filePath := filepath.Join(monitorPath, "movie.mkv")
	writeFileWithMtime(t, filePath, []byte("movie"), clock.Now())
	_, rule := createDirectoryUploadRuleForTest(t, monitorPath)
	rule.ProcessedCacheTTLSeconds = 10

	service := NewService(ServiceOptions{Now: clock.Now})
	service.SetRemoteClient(&fakeRemoteClient{parentID: "remote-root"})
	if err := service.HandleStableFile(context.Background(), rule, filePath); err != nil {
		t.Fatalf("首次处理稳定文件失败: %v", err)
	}
	if err := service.HandleStableFile(context.Background(), rule, filePath); err != nil {
		t.Fatalf("TTL 内重复处理稳定文件失败: %v", err)
	}
	var total int64
	db.Db.Model(&models.DbUploadTask{}).Count(&total)
	if total != 1 {
		t.Fatalf("TTL 内重复处理创建了 %d 条任务，期望 1", total)
	}

	clock.Add(11 * time.Second)
	writeFileWithMtime(t, filePath, []byte("movie changed"), clock.Now())
	if err := service.HandleStableFile(context.Background(), rule, filePath); err != nil {
		t.Fatalf("TTL 过期且签名变化后处理失败: %v", err)
	}
	db.Db.Model(&models.DbUploadTask{}).Count(&total)
	if total != 2 {
		t.Fatalf("TTL 过期且签名变化后任务数 = %d，期望 2", total)
	}
}

func TestServiceSkipsUploadWhenRemoteFileAlreadyExists(t *testing.T) {
	setupDirectoryUploadServiceTestDB(t)
	monitorPath := t.TempDir()
	filePath := filepath.Join(monitorPath, "movie.mkv")
	writeFileWithMtime(t, filePath, []byte("movie"), time.Unix(400, 0))
	_, rule := createDirectoryUploadRuleForTest(t, monitorPath)
	sha1, err := helpers.FileSHA1(filePath)
	if err != nil {
		t.Fatalf("计算测试文件 SHA1 失败: %v", err)
	}
	service := NewService(ServiceOptions{})
	service.SetRemoteClient(&fakeRemoteClient{
		parentID: "remote-root",
		files: map[string]*RemoteFile{
			"movie.mkv": {
				ID:       "remote-file",
				PickCode: "pick-code",
				SHA1:     sha1,
				Size:     int64(len("movie")),
				Mtime:    123,
			},
		},
	})

	if err := service.HandleStableFile(context.Background(), rule, filePath); err != nil {
		t.Fatalf("处理远端已存在文件失败: %v", err)
	}
	var task models.DbUploadTask
	if err := db.Db.First(&task).Error; err != nil {
		t.Fatalf("读取上传任务失败: %v", err)
	}
	if task.Status != models.UploadStatusCompleted ||
		task.UploadResult != models.UploadResultRemoteExists ||
		task.CompletedRemoteFileId != "remote-file" ||
		task.CompletedPickCode != "pick-code" {
		t.Fatalf("上传任务 = %+v，期望远端已存在完成任务", task)
	}
	var strmTask models.StrmGenerationTask
	if err := db.Db.First(&strmTask).Error; err != nil {
		t.Fatalf("读取 STRM 任务失败: %v", err)
	}
	if strmTask.UploadTaskId != task.ID || strmTask.Source != models.StrmGenerationSourceRemoteExists {
		t.Fatalf("STRM 任务 = %+v，期望关联远端已存在上传任务", strmTask)
	}
}
