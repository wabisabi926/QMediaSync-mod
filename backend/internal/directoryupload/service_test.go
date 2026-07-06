package directoryupload

import (
	"context"
	"errors"
	"io"
	"log"
	"os"
	"path/filepath"
	"reflect"
	"sync"
	"testing"
	"time"

	"qmediasync/internal/db"
	"qmediasync/internal/helpers"
	"qmediasync/internal/models"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

type fakeRemoteClient struct {
	parentID       string
	files          map[string]*RemoteFile
	deletedFileIDs []string
	ensureDirErr   error
	ensureDirCalls int
	findFileCalls  int
}

func (c *fakeRemoteClient) EnsureDir(context.Context, *models.DirectoryUploadRule, string) (RemoteDirectory, error) {
	c.ensureDirCalls++
	if c.ensureDirErr != nil {
		return RemoteDirectory{}, c.ensureDirErr
	}
	return RemoteDirectory{ID: c.parentID, Path: "/remote/movie"}, nil
}

func (c *fakeRemoteClient) FindFile(_ context.Context, _ string, fileName string) (*RemoteFile, error) {
	c.findFileCalls++
	if c.files == nil {
		return nil, nil
	}
	return c.files[fileName], nil
}

func (c *fakeRemoteClient) DeleteFile(_ context.Context, _ string, fileID string) error {
	c.deletedFileIDs = append(c.deletedFileIDs, fileID)
	return nil
}

func setupDirectoryUploadServiceTestDB(t *testing.T) {
	t.Helper()
	if helpers.AppLogger == nil {
		helpers.AppLogger = &helpers.QLogger{Logger: log.New(io.Discard, "", 0)}
	}
	testDB, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "directoryupload.db")), &gorm.Config{})
	if err != nil {
		t.Fatalf("打开测试数据库失败: %v", err)
	}
	db.Db = testDB
	if err := db.Db.AutoMigrate(
		&models.Account{},
		&models.SyncPath{},
		&models.DirectoryUploadRule{},
		&models.DirectoryUploadProcessedFile{},
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

type barrierRemoteClient struct {
	parentID string
	waitFor  int
	release  chan struct{}
	once     sync.Once

	mutex          sync.Mutex
	ensureDirCalls int
	findFileCalls  int
}

func newBarrierRemoteClient(parentID string, waitFor int) *barrierRemoteClient {
	return &barrierRemoteClient{
		parentID: parentID,
		waitFor:  waitFor,
		release:  make(chan struct{}),
	}
}

func (c *barrierRemoteClient) EnsureDir(ctx context.Context, _ *models.DirectoryUploadRule, _ string) (RemoteDirectory, error) {
	c.mutex.Lock()
	c.ensureDirCalls++
	if c.ensureDirCalls >= c.waitFor {
		c.once.Do(func() {
			close(c.release)
		})
	}
	c.mutex.Unlock()

	select {
	case <-c.release:
		return RemoteDirectory{ID: c.parentID, Path: "/remote/movie"}, nil
	case <-ctx.Done():
		return RemoteDirectory{}, ctx.Err()
	}
}

func (c *barrierRemoteClient) FindFile(context.Context, string, string) (*RemoteFile, error) {
	c.mutex.Lock()
	c.findFileCalls++
	c.mutex.Unlock()
	return nil, nil
}

func (c *barrierRemoteClient) DeleteFile(context.Context, string, string) error {
	return nil
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

func setSyncPathMetaExtForTest(t *testing.T, syncPathID uint, metaExt []string) {
	t.Helper()
	encoded := models.SettingStrm{MetaExtArr: metaExt}.EncodeArr()
	if encoded == nil {
		t.Fatal("编码测试元数据扩展名失败")
	}
	if err := db.Db.Model(&models.SyncPath{}).
		Where("id = ?", syncPathID).
		Update("meta_ext", encoded.MetaExt).Error; err != nil {
		t.Fatalf("更新同步目录元数据扩展名失败: %v", err)
	}
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

func TestTrackCandidatePathSkipsNestedFileWhenRecursiveDisabled(t *testing.T) {
	setupDirectoryUploadServiceTestDB(t)
	monitorPath := t.TempDir()
	nested := filepath.Join(monitorPath, "show")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatalf("创建测试子目录失败: %v", err)
	}
	rootFile := filepath.Join(monitorPath, "movie.mkv")
	nestedFile := filepath.Join(nested, "episode.mkv")
	writeFileWithMtime(t, rootFile, []byte("movie"), time.Now())
	writeFileWithMtime(t, nestedFile, []byte("episode"), time.Now())
	_, rule := createDirectoryUploadRuleForTest(t, monitorPath)
	rule.Recursive = false

	service := NewService(ServiceOptions{})
	tests := []struct {
		name         string
		path         string
		wantAccepted bool
	}{
		{
			name:         "根目录文件允许入队",
			path:         rootFile,
			wantAccepted: true,
		},
		{
			name:         "子目录文件不入队",
			path:         nestedFile,
			wantAccepted: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			accepted, err := service.trackCandidatePath(context.Background(), rule, tt.path)
			if err != nil {
				t.Fatalf("处理候选文件失败: %v", err)
			}
			if accepted != tt.wantAccepted {
				t.Fatalf("accepted=%v，期望 %v", accepted, tt.wantAccepted)
			}
		})
	}

	got := service.PendingPaths(rule.ID)
	want := []string{rootFile}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("pending paths=%v，期望只包含根目录文件 %v", got, want)
	}
}

func TestScanRuleUsesUploadMetadataSwitchForMetadataFiles(t *testing.T) {
	tests := []struct {
		name           string
		uploadMetadata bool
		customMetaExt  []string
		files          []string
		wantAccepted   int
		wantPending    []string
	}{
		{
			name:           "关闭上传元数据时只加入视频",
			uploadMetadata: false,
			files:          []string{"movie.mkv", "movie.nfo"},
			wantAccepted:   1,
			wantPending:    []string{"movie.mkv"},
		},
		{
			name:           "开启上传元数据时使用全局元数据扩展名",
			uploadMetadata: true,
			files:          []string{"movie.mkv", "movie.nfo"},
			wantAccepted:   2,
			wantPending:    []string{"movie.mkv", "movie.nfo"},
		},
		{
			name:           "开启上传元数据时自定义元数据扩展名优先",
			uploadMetadata: true,
			customMetaExt:  []string{".poster"},
			files:          []string{"movie.mkv", "movie.nfo", "movie.poster"},
			wantAccepted:   2,
			wantPending:    []string{"movie.mkv", "movie.poster"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setupDirectoryUploadServiceTestDB(t)
			monitorPath := t.TempDir()
			for _, name := range tt.files {
				writeFileWithMtime(t, filepath.Join(monitorPath, name), []byte(name), time.Now())
			}
			syncPath, rule := createDirectoryUploadRuleForTest(t, monitorPath)
			rule.UploadMetadata = tt.uploadMetadata
			if len(tt.customMetaExt) > 0 {
				setSyncPathMetaExtForTest(t, syncPath.ID, tt.customMetaExt)
			}

			service := NewService(ServiceOptions{})
			accepted, err := service.ScanRule(context.Background(), rule)
			if err != nil {
				t.Fatalf("扫描目录失败: %v", err)
			}
			if accepted != tt.wantAccepted {
				t.Fatalf("accepted=%d，期望 %d", accepted, tt.wantAccepted)
			}

			want := make([]string, 0, len(tt.wantPending))
			for _, name := range tt.wantPending {
				want = append(want, filepath.Join(monitorPath, name))
			}
			if got := service.PendingPaths(rule.ID); !reflect.DeepEqual(got, want) {
				t.Fatalf("pending paths=%v，期望 %v", got, want)
			}
		})
	}
}

func TestHandleStableFileUsesUploadMetadataSwitchForMetadataFiles(t *testing.T) {
	tests := []struct {
		name           string
		uploadMetadata bool
		wantTasks      int64
	}{
		{name: "关闭上传元数据时跳过元数据文件", uploadMetadata: false, wantTasks: 0},
		{name: "开启上传元数据时创建元数据上传任务", uploadMetadata: true, wantTasks: 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setupDirectoryUploadServiceTestDB(t)
			monitorPath := t.TempDir()
			filePath := filepath.Join(monitorPath, "movie.nfo")
			writeFileWithMtime(t, filePath, []byte("metadata"), time.Unix(125, 0))
			_, rule := createDirectoryUploadRuleForTest(t, monitorPath)
			rule.UploadMetadata = tt.uploadMetadata
			service := NewService(ServiceOptions{})
			service.SetRemoteClient(&fakeRemoteClient{parentID: "remote-root"})

			if err := service.HandleStableFile(context.Background(), rule, filePath); err != nil {
				t.Fatalf("处理稳定元数据文件失败: %v", err)
			}
			var total int64
			if err := db.Db.Model(&models.DbUploadTask{}).Count(&total).Error; err != nil {
				t.Fatalf("统计上传任务失败: %v", err)
			}
			if total != tt.wantTasks {
				t.Fatalf("上传任务数量 = %d，期望 %d", total, tt.wantTasks)
			}
		})
	}
}

func TestScanRuleStableFilesCreateUploadTasks(t *testing.T) {
	setupDirectoryUploadServiceTestDB(t)
	clock := &fakeClock{now: time.Unix(150, 0)}
	monitorPath := t.TempDir()
	nested := filepath.Join(monitorPath, "show", "Season 01")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatalf("创建测试目录失败: %v", err)
	}
	writeFileWithMtime(t, filepath.Join(monitorPath, "movie.mkv"), []byte("movie"), clock.Now())
	writeFileWithMtime(t, filepath.Join(nested, "episode.mp4"), []byte("episode"), clock.Now())
	writeFileWithMtime(t, filepath.Join(nested, "ignore.tmp"), []byte("tmp"), clock.Now())

	_, rule := createDirectoryUploadRuleForTest(t, monitorPath)
	service := NewService(ServiceOptions{Now: clock.Now})
	service.SetRemoteClient(&fakeRemoteClient{parentID: "remote-root"})
	if accepted, err := service.ScanRule(context.Background(), rule); err != nil || accepted != 2 {
		t.Fatalf("扫描目录 accepted=%d err=%v，期望 2 个候选视频", accepted, err)
	}
	if ready, err := service.CheckStableFiles(rule); err != nil || len(ready) != 0 {
		t.Fatalf("首次稳定性检查 ready=%v err=%v，期望未就绪", ready, err)
	}
	clock.Add(15 * time.Second)
	for i := 1; i < 3; i++ {
		ready, err := service.CheckStableFiles(rule)
		if err != nil {
			t.Fatalf("第 %d 次稳定性检查失败: %v", i, err)
		}
		if len(ready) != 0 {
			t.Fatalf("第 %d 次稳定性检查 ready=%v，期望未就绪", i, ready)
		}
	}
	ready, err := service.CheckStableFiles(rule)
	if err != nil {
		t.Fatalf("第三次稳定性检查失败: %v", err)
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
		info, err := os.Stat(task.LocalFullPath)
		if err != nil {
			t.Fatalf("读取任务源文件失败: %v", err)
		}
		if task.FileSize != info.Size() {
			t.Fatalf("file_size = %d，期望 %d", task.FileSize, info.Size())
		}
		if task.LocalMtimeNs != info.ModTime().UnixNano() {
			t.Fatalf("local_mtime_ns = %d，期望 %d", task.LocalMtimeNs, info.ModTime().UnixNano())
		}
		expectedFingerprint := models.BuildDirectoryUploadSourceFingerprint(info.Size(), info.ModTime().UnixNano())
		if task.SourceFingerprint != expectedFingerprint {
			t.Fatalf("source_fingerprint = %q，期望 %q", task.SourceFingerprint, expectedFingerprint)
		}
	}
}

func TestProcessStableFilesRequeuesFileWhenCreatingUploadTaskFails(t *testing.T) {
	setupDirectoryUploadServiceTestDB(t)
	clock := &fakeClock{now: time.Unix(200, 0)}
	monitorPath := t.TempDir()
	filePath := filepath.Join(monitorPath, "movie.mkv")
	writeFileWithMtime(t, filePath, []byte("movie"), clock.Now())
	_, rule := createDirectoryUploadRuleForTest(t, monitorPath)
	service := NewService(ServiceOptions{Now: clock.Now})
	service.SetRemoteClient(&fakeRemoteClient{
		parentID:     "remote-root",
		ensureDirErr: errors.New("temporary remote error"),
	})

	if accepted, err := service.ScanRule(context.Background(), rule); err != nil || accepted != 1 {
		t.Fatalf("扫描目录 accepted=%d err=%v，期望 1 个候选视频", accepted, err)
	}
	if ready, err := service.CheckStableFiles(rule); err != nil || len(ready) != 0 {
		t.Fatalf("首次稳定性检查 ready=%v err=%v，期望未就绪", ready, err)
	}
	clock.Add(15 * time.Second)
	for i := 1; i < 3; i++ {
		ready, err := service.CheckStableFiles(rule)
		if err != nil {
			t.Fatalf("第 %d 次稳定性检查失败: %v", i, err)
		}
		if len(ready) != 0 {
			t.Fatalf("第 %d 次稳定性检查 ready=%v，期望未就绪", i, ready)
		}
	}

	service.processStableFiles(context.Background(), rule)

	got := service.PendingPaths(rule.ID)
	want := []string{filePath}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("pending paths=%v，期望失败后重新加入稳定性队列 %v", got, want)
	}
	var total int64
	if err := db.Db.Model(&models.DbUploadTask{}).Count(&total).Error; err != nil {
		t.Fatalf("统计上传任务失败: %v", err)
	}
	if total != 0 {
		t.Fatalf("上传任务数量 = %d，期望创建任务失败时不落库", total)
	}
}

func TestProcessStableFilesDoesNotRequeueRemoteConflict(t *testing.T) {
	setupDirectoryUploadServiceTestDB(t)
	clock := &fakeClock{now: time.Unix(250, 0)}
	monitorPath := t.TempDir()
	filePath := filepath.Join(monitorPath, "movie.mkv")
	writeFileWithMtime(t, filePath, []byte("movie"), clock.Now())
	_, rule := createDirectoryUploadRuleForTest(t, monitorPath)
	rule.OverwriteMode = models.DirectoryUploadOverwriteFailConflict
	service := NewService(ServiceOptions{Now: clock.Now})
	service.SetRemoteClient(&fakeRemoteClient{
		parentID: "remote-root",
		files: map[string]*RemoteFile{
			"movie.mkv": {ID: "remote-file", PickCode: "pick-code", SHA1: "different", Size: 999},
		},
	})

	if accepted, err := service.ScanRule(context.Background(), rule); err != nil || accepted != 1 {
		t.Fatalf("扫描目录 accepted=%d err=%v，期望 1 个候选视频", accepted, err)
	}
	if ready, err := service.CheckStableFiles(rule); err != nil || len(ready) != 0 {
		t.Fatalf("首次稳定性检查 ready=%v err=%v，期望未就绪", ready, err)
	}
	clock.Add(15 * time.Second)
	for i := 1; i < 3; i++ {
		ready, err := service.CheckStableFiles(rule)
		if err != nil {
			t.Fatalf("第 %d 次稳定性检查失败: %v", i, err)
		}
		if len(ready) != 0 {
			t.Fatalf("第 %d 次稳定性检查 ready=%v，期望未就绪", i, ready)
		}
	}

	service.processStableFiles(context.Background(), rule)

	if got := service.PendingPaths(rule.ID); len(got) != 0 {
		t.Fatalf("pending paths=%v，远端冲突属于确定性失败，不应重新加入稳定性队列", got)
	}
	var total int64
	if err := db.Db.Model(&models.DbUploadTask{}).Count(&total).Error; err != nil {
		t.Fatalf("统计上传任务失败: %v", err)
	}
	if total != 0 {
		t.Fatalf("上传任务数量 = %d，期望远端冲突不创建上传任务", total)
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
	if err := db.Db.Model(&models.DbUploadTask{}).Where("local_full_path = ?", filePath).Update("status", models.UploadStatusCompleted).Error; err != nil {
		t.Fatalf("标记已有上传任务完成失败: %v", err)
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

func TestHandleStableFileSkipsProcessedRetainedSourceAfterServiceRestart(t *testing.T) {
	setupDirectoryUploadServiceTestDB(t)
	clock := &fakeClock{now: time.Unix(320, 0)}
	monitorPath := t.TempDir()
	filePath := filepath.Join(monitorPath, "movie.mkv")
	writeFileWithMtime(t, filePath, []byte("movie"), clock.Now())
	_, rule := createDirectoryUploadRuleForTest(t, monitorPath)

	firstService := NewService(ServiceOptions{Now: clock.Now})
	firstRemote := &fakeRemoteClient{parentID: "remote-root"}
	firstService.SetRemoteClient(firstRemote)
	if err := firstService.HandleStableFile(context.Background(), rule, filePath); err != nil {
		t.Fatalf("首次处理稳定文件失败: %v", err)
	}
	var task models.DbUploadTask
	if err := db.Db.Where("local_full_path = ?", filePath).First(&task).Error; err != nil {
		t.Fatalf("读取首次上传任务失败: %v", err)
	}
	if err := db.Db.Model(&models.DbUploadTask{}).
		Where("id = ?", task.ID).
		Update("status", models.UploadStatusCompleted).Error; err != nil {
		t.Fatalf("标记首次上传任务完成失败: %v", err)
	}
	if err := models.MarkDirectoryUploadProcessedUploaded(task.ID, models.DirectoryUploadProcessedResultUploaded); err != nil {
		t.Fatalf("标记 processed 上传完成失败: %v", err)
	}

	clock.Add(time.Hour)
	restartedService := NewService(ServiceOptions{Now: clock.Now})
	restartedRemote := &fakeRemoteClient{parentID: "remote-root"}
	restartedService.SetRemoteClient(restartedRemote)
	if err := restartedService.HandleStableFile(context.Background(), rule, filePath); err != nil {
		t.Fatalf("服务重启后处理已完成源文件失败: %v", err)
	}

	var total int64
	if err := db.Db.Model(&models.DbUploadTask{}).
		Where("source = ? AND local_full_path = ?", models.UploadSourceDirectoryMonitor, filePath).
		Count(&total).Error; err != nil {
		t.Fatalf("统计上传任务失败: %v", err)
	}
	if total != 1 {
		t.Fatalf("上传任务数量 = %d，期望服务重启后保留源文件不重复入队", total)
	}
	if restartedRemote.findFileCalls != 0 || restartedRemote.ensureDirCalls != 0 {
		t.Fatalf("远端调用 ensure=%d find=%d，期望命中 processed 表后不访问远端", restartedRemote.ensureDirCalls, restartedRemote.findFileCalls)
	}

	scopeHash := models.BuildDirectoryUploadScopeHash(rule)
	sourceKey := models.BuildDirectoryUploadSourceKey(scopeHash, "movie.mkv")
	processed, err := models.FindDirectoryUploadProcessedBySourceKey(sourceKey)
	if err != nil {
		t.Fatalf("读取 processed 记录失败: %v", err)
	}
	if processed.LastSeenAt != clock.Now().Unix() {
		t.Fatalf("last_seen_at = %d，期望服务重启后跳过时更新为 %d", processed.LastSeenAt, clock.Now().Unix())
	}
}

func TestHandleStableFileProcessesAgainWhenSourceFingerprintChanges(t *testing.T) {
	setupDirectoryUploadServiceTestDB(t)
	clock := &fakeClock{now: time.Unix(330, 0)}
	monitorPath := t.TempDir()
	filePath := filepath.Join(monitorPath, "movie.mkv")
	writeFileWithMtime(t, filePath, []byte("movie"), clock.Now())
	_, rule := createDirectoryUploadRuleForTest(t, monitorPath)

	firstService := NewService(ServiceOptions{Now: clock.Now})
	firstService.SetRemoteClient(&fakeRemoteClient{parentID: "remote-root"})
	if err := firstService.HandleStableFile(context.Background(), rule, filePath); err != nil {
		t.Fatalf("首次处理稳定文件失败: %v", err)
	}
	var firstTask models.DbUploadTask
	if err := db.Db.Where("local_full_path = ?", filePath).First(&firstTask).Error; err != nil {
		t.Fatalf("读取首次上传任务失败: %v", err)
	}
	if err := db.Db.Model(&models.DbUploadTask{}).
		Where("id = ?", firstTask.ID).
		Update("status", models.UploadStatusCompleted).Error; err != nil {
		t.Fatalf("标记首次上传任务完成失败: %v", err)
	}
	if err := models.MarkDirectoryUploadProcessedUploaded(firstTask.ID, models.DirectoryUploadProcessedResultUploaded); err != nil {
		t.Fatalf("标记 processed 上传完成失败: %v", err)
	}

	clock.Add(time.Hour)
	writeFileWithMtime(t, filePath, []byte("movie changed"), clock.Now())
	restartedService := NewService(ServiceOptions{Now: clock.Now})
	restartedService.SetRemoteClient(&fakeRemoteClient{parentID: "remote-root"})
	if err := restartedService.HandleStableFile(context.Background(), rule, filePath); err != nil {
		t.Fatalf("源文件 fingerprint 变化后处理失败: %v", err)
	}

	var tasks []models.DbUploadTask
	if err := db.Db.Where("source = ? AND local_full_path = ?", models.UploadSourceDirectoryMonitor, filePath).
		Order("id ASC").
		Find(&tasks).Error; err != nil {
		t.Fatalf("读取上传任务失败: %v", err)
	}
	if len(tasks) != 2 {
		t.Fatalf("上传任务数量 = %d，期望 fingerprint 变化后重新创建任务", len(tasks))
	}
	if tasks[0].SourceFingerprint == tasks[1].SourceFingerprint {
		t.Fatalf("两次任务 source_fingerprint 相同：%q，期望文件变化后不同", tasks[0].SourceFingerprint)
	}

	scopeHash := models.BuildDirectoryUploadScopeHash(rule)
	sourceKey := models.BuildDirectoryUploadSourceKey(scopeHash, "movie.mkv")
	processed, err := models.FindDirectoryUploadProcessedBySourceKey(sourceKey)
	if err != nil {
		t.Fatalf("读取 processed 记录失败: %v", err)
	}
	if processed.UploadTaskId != tasks[1].ID ||
		processed.Result != models.DirectoryUploadProcessedResultQueued ||
		processed.SourceFingerprint != tasks[1].SourceFingerprint {
		t.Fatalf("processed 记录 = %+v，期望更新为第二次 queued 任务", processed)
	}
}

func TestHandleStableFileScopeChangeBypassesOldTerminalMemoryCache(t *testing.T) {
	setupDirectoryUploadServiceTestDB(t)
	clock := &fakeClock{now: time.Unix(332, 0)}
	monitorPath := t.TempDir()
	filePath := filepath.Join(monitorPath, "movie.mkv")
	writeFileWithMtime(t, filePath, []byte("movie"), clock.Now())
	_, rule := createDirectoryUploadRuleForTest(t, monitorPath)
	rule.RemoteRootPath = "/remote/old"

	info, err := os.Stat(filePath)
	if err != nil {
		t.Fatalf("读取测试文件失败: %v", err)
	}
	oldScopeHash := models.BuildDirectoryUploadScopeHash(rule)
	oldSourceKey := models.BuildDirectoryUploadSourceKey(oldScopeHash, "movie.mkv")
	fingerprint := models.BuildDirectoryUploadSourceFingerprint(info.Size(), info.ModTime().UnixNano())
	if err := db.Db.Create(&models.DirectoryUploadProcessedFile{
		RuleId:            rule.ID,
		SyncPathId:        rule.SyncPathId,
		AccountId:         rule.AccountId,
		ScopeHash:         oldScopeHash,
		SourceKey:         oldSourceKey,
		RelativePath:      "movie.mkv",
		LocalFullPath:     filePath,
		SourceFingerprint: fingerprint,
		FileSize:          info.Size(),
		LocalMtimeNs:      info.ModTime().UnixNano(),
		Result:            models.DirectoryUploadProcessedResultUploaded,
		ProcessedAt:       clock.Now().Unix(),
		LastSeenAt:        clock.Now().Unix(),
	}).Error; err != nil {
		t.Fatalf("创建旧范围 processed 记录失败: %v", err)
	}

	service := NewService(ServiceOptions{Now: clock.Now})
	remote := &fakeRemoteClient{parentID: "remote-root"}
	service.SetRemoteClient(remote)
	if err := service.HandleStableFile(context.Background(), rule, filePath); err != nil {
		t.Fatalf("旧范围终态缓存预热失败: %v", err)
	}
	if remote.ensureDirCalls != 0 || remote.findFileCalls != 0 {
		t.Fatalf("旧范围命中 processed 后不应访问远端，ensure=%d find=%d", remote.ensureDirCalls, remote.findFileCalls)
	}

	rule.RemoteRootPath = "/remote/new"
	if err := service.HandleStableFile(context.Background(), rule, filePath); err != nil {
		t.Fatalf("范围变更后处理稳定文件失败: %v", err)
	}

	var task models.DbUploadTask
	if err := db.Db.Where("source = ? AND local_full_path = ?", models.UploadSourceDirectoryMonitor, filePath).First(&task).Error; err != nil {
		t.Fatalf("范围变更后应创建新上传任务: %v", err)
	}
	if task.RemoteFileId != "/remote/new/movie.mkv" {
		t.Fatalf("新上传任务 remote_file_id = %q，期望使用新远端范围", task.RemoteFileId)
	}
	newScopeHash := models.BuildDirectoryUploadScopeHash(rule)
	newSourceKey := models.BuildDirectoryUploadSourceKey(newScopeHash, "movie.mkv")
	processed, err := models.FindDirectoryUploadProcessedBySourceKey(newSourceKey)
	if err != nil {
		t.Fatalf("读取新范围 processed 记录失败: %v", err)
	}
	if processed.Result != models.DirectoryUploadProcessedResultQueued || processed.UploadTaskId != task.ID {
		t.Fatalf("新范围 processed 记录 = %+v，期望关联新 queued 任务 %+v", processed, task)
	}
}

func TestHandleStableFileQueuedAllowsRetryWhenTaskInactiveWithinTTL(t *testing.T) {
	tests := []struct {
		name       string
		status     models.UploadStatus
		deleteTask bool
	}{
		{name: "关联任务失败", status: models.UploadStatusFailed},
		{name: "关联任务已取消", status: models.UploadStatusCancelled},
		{name: "关联任务已完成", status: models.UploadStatusCompleted},
		{name: "关联任务不存在", status: models.UploadStatusFailed, deleteTask: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setupDirectoryUploadServiceTestDB(t)
			clock := &fakeClock{now: time.Unix(335, 0)}
			monitorPath := t.TempDir()
			filePath := filepath.Join(monitorPath, "movie.mkv")
			writeFileWithMtime(t, filePath, []byte("movie"), clock.Now())
			_, rule := createDirectoryUploadRuleForTest(t, monitorPath)
			rule.ProcessedCacheTTLSeconds = 600

			service := NewService(ServiceOptions{Now: clock.Now})
			service.SetRemoteClient(&fakeRemoteClient{parentID: "remote-root"})
			if err := service.HandleStableFile(context.Background(), rule, filePath); err != nil {
				t.Fatalf("首次处理稳定文件失败: %v", err)
			}
			var firstTask models.DbUploadTask
			if err := db.Db.Where("local_full_path = ?", filePath).First(&firstTask).Error; err != nil {
				t.Fatalf("读取首次上传任务失败: %v", err)
			}
			if tt.deleteTask {
				if err := db.Db.Delete(&firstTask).Error; err != nil {
					t.Fatalf("删除首次上传任务失败: %v", err)
				}
			} else if err := db.Db.Model(&models.DbUploadTask{}).
				Where("id = ?", firstTask.ID).
				Update("status", tt.status).Error; err != nil {
				t.Fatalf("更新首次上传任务状态失败: %v", err)
			}

			if err := service.HandleStableFile(context.Background(), rule, filePath); err != nil {
				t.Fatalf("TTL 内处理非活跃 queued 源文件失败: %v", err)
			}

			var latestTask models.DbUploadTask
			if err := db.Db.Where("source = ? AND local_full_path = ?", models.UploadSourceDirectoryMonitor, filePath).
				Order("id DESC").
				First(&latestTask).Error; err != nil {
				t.Fatalf("读取重新创建的上传任务失败: %v", err)
			}
			if latestTask.ID == firstTask.ID {
				t.Fatalf("最新上传任务 ID = %d，期望 TTL 内非活跃 queued 允许重新创建任务", latestTask.ID)
			}

			scopeHash := models.BuildDirectoryUploadScopeHash(rule)
			sourceKey := models.BuildDirectoryUploadSourceKey(scopeHash, "movie.mkv")
			processed, err := models.FindDirectoryUploadProcessedBySourceKey(sourceKey)
			if err != nil {
				t.Fatalf("读取 processed 记录失败: %v", err)
			}
			if processed.Result != models.DirectoryUploadProcessedResultQueued ||
				processed.UploadTaskId != latestTask.ID ||
				processed.SourceFingerprint != latestTask.SourceFingerprint {
				t.Fatalf("processed 记录 = %+v，期望关联重新创建的 queued 任务 %+v", processed, latestTask)
			}
		})
	}
}

func TestHandleStableFileConcurrentSameSourceCreatesOneTask(t *testing.T) {
	setupDirectoryUploadServiceTestDB(t)
	clock := &fakeClock{now: time.Unix(340, 0)}
	monitorPath := t.TempDir()
	filePath := filepath.Join(monitorPath, "movie.mkv")
	writeFileWithMtime(t, filePath, []byte("movie"), clock.Now())
	_, rule := createDirectoryUploadRuleForTest(t, monitorPath)

	const workers = 8
	remote := newBarrierRemoteClient("remote-root", workers)
	services := make([]*Service, 0, workers)
	for range workers {
		service := NewService(ServiceOptions{Now: clock.Now})
		service.SetRemoteClient(remote)
		services = append(services, service)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	start := make(chan struct{})
	errCh := make(chan error, workers)
	var wg sync.WaitGroup
	for _, service := range services {
		wg.Add(1)
		go func(service *Service) {
			defer wg.Done()
			<-start
			errCh <- service.HandleStableFile(ctx, rule, filePath)
		}(service)
	}
	close(start)
	wg.Wait()
	close(errCh)
	for err := range errCh {
		if err != nil {
			t.Fatalf("并发处理稳定文件失败: %v", err)
		}
	}

	var tasks []models.DbUploadTask
	if err := db.Db.Where("source = ? AND local_full_path = ?", models.UploadSourceDirectoryMonitor, filePath).
		Order("id ASC").
		Find(&tasks).Error; err != nil {
		t.Fatalf("读取上传任务失败: %v", err)
	}
	if len(tasks) != 1 {
		t.Fatalf("上传任务数量 = %d，期望并发同一稳定文件只创建 1 个任务: %+v", len(tasks), tasks)
	}
	scopeHash := models.BuildDirectoryUploadScopeHash(rule)
	sourceKey := models.BuildDirectoryUploadSourceKey(scopeHash, "movie.mkv")
	processed, err := models.FindDirectoryUploadProcessedBySourceKey(sourceKey)
	if err != nil {
		t.Fatalf("读取 processed 记录失败: %v", err)
	}
	if processed.Result != models.DirectoryUploadProcessedResultQueued ||
		processed.UploadTaskId != tasks[0].ID ||
		processed.SourceFingerprint != tasks[0].SourceFingerprint {
		t.Fatalf("processed 记录 = %+v，期望 queued 且关联唯一上传任务 %+v", processed, tasks[0])
	}
	if remote.ensureDirCalls != workers {
		t.Fatalf("ensureDir 调用数 = %d，期望 %d 个并发调用都进入创建路径", remote.ensureDirCalls, workers)
	}
}

func TestServiceSkipsStableFileWhenActiveUploadTaskExists(t *testing.T) {
	cases := []struct {
		name   string
		status models.UploadStatus
	}{
		{name: "已有等待任务", status: models.UploadStatusPending},
		{name: "已有上传中任务", status: models.UploadStatusUploading},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			setupDirectoryUploadServiceTestDB(t)
			monitorPath := t.TempDir()
			filePath := filepath.Join(monitorPath, "movie.mkv")
			writeFileWithMtime(t, filePath, []byte("movie"), time.Unix(350, 0))
			_, rule := createDirectoryUploadRuleForTest(t, monitorPath)
			existing := &models.DbUploadTask{
				Source:        models.UploadSourceDirectoryMonitor,
				AccountId:     rule.AccountId,
				SyncPathId:    rule.SyncPathId,
				SourceType:    models.SourceType115,
				LocalFullPath: filePath,
				RemoteFileId:  "/remote/movie.mkv",
				FileName:      "movie.mkv",
				Status:        tt.status,
				FileSize:      int64(len("movie")),
				UploadResult:  models.UploadResultUnknown,
			}
			if err := db.Db.Create(existing).Error; err != nil {
				t.Fatalf("创建已有上传任务失败: %v", err)
			}

			service := NewService(ServiceOptions{})
			service.SetRemoteClient(&fakeRemoteClient{parentID: "remote-root"})
			if err := service.HandleStableFile(context.Background(), rule, filePath); err != nil {
				t.Fatalf("处理已有上传任务的稳定文件失败: %v", err)
			}

			var total int64
			if err := db.Db.Model(&models.DbUploadTask{}).
				Where("source = ? AND local_full_path = ?", models.UploadSourceDirectoryMonitor, filePath).
				Count(&total).Error; err != nil {
				t.Fatalf("统计上传任务失败: %v", err)
			}
			if total != 1 {
				t.Fatalf("上传任务数量 = %d，期望跳过重复入队保持 1 条", total)
			}
		})
	}
}

func TestServiceSkipsUploadWhenRemoteFileAlreadyExists(t *testing.T) {
	setupDirectoryUploadServiceTestDB(t)
	clock := &fakeClock{now: time.Unix(400, 0)}
	monitorPath := t.TempDir()
	filePath := filepath.Join(monitorPath, "movie.mkv")
	writeFileWithMtime(t, filePath, []byte("movie"), clock.Now())
	_, rule := createDirectoryUploadRuleForTest(t, monitorPath)
	sha1, err := helpers.FileSHA1(filePath)
	if err != nil {
		t.Fatalf("计算测试文件 SHA1 失败: %v", err)
	}
	service := NewService(ServiceOptions{Now: clock.Now})
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
	info, err := os.Stat(filePath)
	if err != nil {
		t.Fatalf("读取任务源文件失败: %v", err)
	}
	if task.FileSize != info.Size() {
		t.Fatalf("file_size = %d，期望 %d", task.FileSize, info.Size())
	}
	if task.LocalMtimeNs != info.ModTime().UnixNano() {
		t.Fatalf("local_mtime_ns = %d，期望 %d", task.LocalMtimeNs, info.ModTime().UnixNano())
	}
	expectedFingerprint := models.BuildDirectoryUploadSourceFingerprint(info.Size(), info.ModTime().UnixNano())
	if task.SourceFingerprint != expectedFingerprint {
		t.Fatalf("source_fingerprint = %q，期望 %q", task.SourceFingerprint, expectedFingerprint)
	}
	var strmTask models.StrmGenerationTask
	if err := db.Db.First(&strmTask).Error; err != nil {
		t.Fatalf("读取 STRM 任务失败: %v", err)
	}
	if strmTask.UploadTaskId != task.ID || strmTask.Source != models.StrmGenerationSourceRemoteExists {
		t.Fatalf("STRM 任务 = %+v，期望关联远端已存在上传任务", strmTask)
	}
}

func TestServiceSkipsRemoteConflictWithoutUpload(t *testing.T) {
	setupDirectoryUploadServiceTestDB(t)
	monitorPath := t.TempDir()
	filePath := filepath.Join(monitorPath, "movie.mkv")
	writeFileWithMtime(t, filePath, []byte("movie"), time.Unix(450, 0))
	_, rule := createDirectoryUploadRuleForTest(t, monitorPath)
	rule.OverwriteMode = models.DirectoryUploadOverwriteSkipSame

	remoteClient := &fakeRemoteClient{
		parentID: "remote-root",
		files: map[string]*RemoteFile{
			"movie.mkv": {ID: "remote-file", PickCode: "pick-code", SHA1: "different", Size: int64(len("movie"))},
		},
	}
	service := NewService(ServiceOptions{})
	service.SetRemoteClient(remoteClient)

	if err := service.HandleStableFile(context.Background(), rule, filePath); err != nil {
		t.Fatalf("跳过远端同名不同文件失败: %v", err)
	}
	var total int64
	db.Db.Model(&models.DbUploadTask{}).Count(&total)
	if total != 0 {
		t.Fatalf("上传任务数量 = %d，期望 0", total)
	}
	if len(remoteClient.deletedFileIDs) != 0 {
		t.Fatalf("删除远端文件 = %v，期望不删除", remoteClient.deletedFileIDs)
	}
	if err := service.HandleStableFile(context.Background(), rule, filePath); err != nil {
		t.Fatalf("已跳过文件在缓存期内重复处理失败: %v", err)
	}
	db.Db.Model(&models.DbUploadTask{}).Count(&total)
	if total != 0 {
		t.Fatalf("重复处理后的上传任务数量 = %d，期望 0", total)
	}
}

func TestServiceStopsWhenRemoteConflictExists(t *testing.T) {
	setupDirectoryUploadServiceTestDB(t)
	monitorPath := t.TempDir()
	filePath := filepath.Join(monitorPath, "movie.mkv")
	writeFileWithMtime(t, filePath, []byte("movie"), time.Unix(500, 0))
	_, rule := createDirectoryUploadRuleForTest(t, monitorPath)
	rule.OverwriteMode = models.DirectoryUploadOverwriteFailConflict

	service := NewService(ServiceOptions{})
	service.SetRemoteClient(&fakeRemoteClient{
		parentID: "remote-root",
		files: map[string]*RemoteFile{
			"movie.mkv": {ID: "remote-file", PickCode: "pick-code", SHA1: "different", Size: 999},
		},
	})

	err := service.HandleStableFile(context.Background(), rule, filePath)
	if err == nil {
		t.Fatal("远端同名不同文件时应停止创建上传任务")
	}
	var total int64
	db.Db.Model(&models.DbUploadTask{}).Count(&total)
	if total != 0 {
		t.Fatalf("上传任务数量 = %d，期望 0", total)
	}
}

func TestServiceReplacesRemoteConflictBeforeUpload(t *testing.T) {
	setupDirectoryUploadServiceTestDB(t)
	monitorPath := t.TempDir()
	filePath := filepath.Join(monitorPath, "movie.mkv")
	writeFileWithMtime(t, filePath, []byte("movie"), time.Unix(600, 0))
	_, rule := createDirectoryUploadRuleForTest(t, monitorPath)
	rule.OverwriteMode = models.DirectoryUploadOverwriteReplaceConflict

	remoteClient := &fakeRemoteClient{
		parentID: "remote-root",
		files: map[string]*RemoteFile{
			"movie.mkv": {ID: "remote-file", PickCode: "pick-code", SHA1: "different", Size: 999},
		},
	}
	service := NewService(ServiceOptions{})
	service.SetRemoteClient(remoteClient)

	if err := service.HandleStableFile(context.Background(), rule, filePath); err != nil {
		t.Fatalf("覆盖远端同名文件后创建上传任务失败: %v", err)
	}
	if !reflect.DeepEqual(remoteClient.deletedFileIDs, []string{"remote-file"}) {
		t.Fatalf("删除远端文件 = %v，期望 [remote-file]", remoteClient.deletedFileIDs)
	}
	var task models.DbUploadTask
	if err := db.Db.First(&task).Error; err != nil {
		t.Fatalf("读取上传任务失败: %v", err)
	}
	if task.Status != models.UploadStatusPending || task.UploadResult != models.UploadResultUnknown {
		t.Fatalf("上传任务 = %+v，期望覆盖后创建 pending 任务", task)
	}
}
