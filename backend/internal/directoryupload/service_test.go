package directoryupload

import (
	"context"
	"errors"
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
	parentID       string
	files          map[string]*RemoteFile
	deletedFileIDs []string
	ensureDirErr   error
}

func (c *fakeRemoteClient) EnsureDir(context.Context, *models.DirectoryUploadRule, string) (RemoteDirectory, error) {
	if c.ensureDirErr != nil {
		return RemoteDirectory{}, c.ensureDirErr
	}
	return RemoteDirectory{ID: c.parentID, Path: "/remote/movie"}, nil
}

func (c *fakeRemoteClient) FindFile(_ context.Context, _ string, fileName string) (*RemoteFile, error) {
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
