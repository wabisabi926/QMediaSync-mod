package models

import (
	"io"
	"log"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"

	"Q115-STRM/internal/db"
	"Q115-STRM/internal/helpers"
)

func TestBatchCreateTableCreatesMigratorTable(t *testing.T) {
	testDb, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("打开测试数据库失败: %v", err)
	}
	db.Db = testDb

	if err := BatchCreateTable(); err != nil {
		t.Fatalf("批量建表失败: %v", err)
	}
	if !db.Db.Migrator().HasTable(Migrator{}) {
		t.Fatal("批量建表应创建 migrator 表")
	}
}

func TestBatchCreateTableCreatesEmbyLibraryRefreshTasksTable(t *testing.T) {
	testDb, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("打开测试数据库失败: %v", err)
	}
	db.Db = testDb

	if err := BatchCreateTable(); err != nil {
		t.Fatalf("批量建表失败: %v", err)
	}
	if !db.Db.Migrator().HasTable(EmbyLibraryRefreshTask{}) {
		t.Fatal("批量建表应创建 emby_library_refresh_tasks 表")
	}
}

func setupMigratorVersion43TestDB(t *testing.T) {
	t.Helper()
	if helpers.AppLogger == nil {
		helpers.AppLogger = &helpers.QLogger{
			Logger: log.New(io.Discard, "", 0),
		}
	}
	testDb, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("打开测试数据库失败: %v", err)
	}
	db.Db = testDb
	if err := db.Db.AutoMigrate(&Migrator{}, &DbDownloadTask{}, &DbUploadTask{}); err != nil {
		t.Fatalf("创建测试表失败: %v", err)
	}
	if err := db.Db.Create(&Migrator{VersionCode: 43}).Error; err != nil {
		t.Fatalf("创建迁移版本记录失败: %v", err)
	}
}

func TestMigrateTaskSourceEnumValues(t *testing.T) {
	setupMigratorVersion43TestDB(t)

	downloadTasks := []DbDownloadTask{
		{RemoteFileId: "download-strm", Source: DownloadSource("strm同步"), SourceType: SourceType115},
		{RemoteFileId: "download-local", Source: DownloadSource("本地文件"), SourceType: SourceTypeLocal},
		{RemoteFileId: "download-emby", Source: DownloadSource("emby媒体信息提取"), SourceType: SourceType("emby媒体信息提取")},
		{RemoteFileId: "download-already-new", Source: DownloadSource("strm_sync"), SourceType: SourceType115},
		{RemoteFileId: "download-unknown", Source: DownloadSource("custom_source"), SourceType: SourceType("custom_type")},
	}
	if err := db.Db.Create(&downloadTasks).Error; err != nil {
		t.Fatalf("创建下载任务测试数据失败: %v", err)
	}

	uploadTasks := []DbUploadTask{
		{RemoteFileId: "upload-strm", LocalFullPath: "/tmp/strm.nfo", Source: UploadSource("strm同步"), SourceType: SourceType115},
		{RemoteFileId: "upload-scrape", LocalFullPath: "/tmp/scrape.nfo", Source: UploadSource("刮削整理"), SourceType: SourceType115},
		{RemoteFileId: "upload-already-new", LocalFullPath: "/tmp/already-new.nfo", Source: UploadSource("strm_sync"), SourceType: SourceType115},
		{RemoteFileId: "upload-unknown", LocalFullPath: "/tmp/unknown.nfo", Source: UploadSource("custom_source"), SourceType: SourceType("custom_type")},
	}
	if err := db.Db.Create(&uploadTasks).Error; err != nil {
		t.Fatalf("创建上传任务测试数据失败: %v", err)
	}

	Migrate()

	var migrator Migrator
	if err := db.Db.First(&migrator).Error; err != nil {
		t.Fatalf("读取迁移版本失败: %v", err)
	}
	if migrator.VersionCode != 44 {
		t.Fatalf("迁移版本 = %d，期望 44", migrator.VersionCode)
	}

	assertDownloadTaskSource(t, "download-strm", "strm_sync", "115")
	assertDownloadTaskSource(t, "download-local", "local_file", "local")
	assertDownloadTaskSource(t, "download-emby", "emby_media", "emby_media")
	assertDownloadTaskSource(t, "download-already-new", "strm_sync", "115")
	assertDownloadTaskSource(t, "download-unknown", "custom_source", "custom_type")
	assertUploadTaskSource(t, "upload-strm", "strm_sync")
	assertUploadTaskSource(t, "upload-scrape", "scrape_organize")
	assertUploadTaskSource(t, "upload-already-new", "strm_sync")
	assertUploadTaskSource(t, "upload-unknown", "custom_source")
}

func assertDownloadTaskSource(t *testing.T, remoteFileId string, wantSource string, wantSourceType string) {
	t.Helper()
	var task DbDownloadTask
	if err := db.Db.Where("remote_file_id = ?", remoteFileId).First(&task).Error; err != nil {
		t.Fatalf("读取下载任务 %s 失败: %v", remoteFileId, err)
	}
	if string(task.Source) != wantSource {
		t.Fatalf("下载任务 %s source = %s，期望 %s", remoteFileId, task.Source, wantSource)
	}
	if string(task.SourceType) != wantSourceType {
		t.Fatalf("下载任务 %s source_type = %s，期望 %s", remoteFileId, task.SourceType, wantSourceType)
	}
}

func assertUploadTaskSource(t *testing.T, remoteFileId string, wantSource string) {
	t.Helper()
	var task DbUploadTask
	if err := db.Db.Where("remote_file_id = ?", remoteFileId).First(&task).Error; err != nil {
		t.Fatalf("读取上传任务 %s 失败: %v", remoteFileId, err)
	}
	if string(task.Source) != wantSource {
		t.Fatalf("上传任务 %s source = %s，期望 %s", remoteFileId, task.Source, wantSource)
	}
}
