package models

import (
	"io"
	"log"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"

	"qmediasync/internal/db"
	"qmediasync/internal/helpers"
	"qmediasync/internal/notification"
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

func TestBatchCreateTableCreatesUploadStrmTables(t *testing.T) {
	testDb, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("打开测试数据库失败: %v", err)
	}
	db.Db = testDb

	if err := BatchCreateTable(); err != nil {
		t.Fatalf("批量建表失败: %v", err)
	}
	for _, table := range []any{
		UploadSession{},
		DirectoryUploadRule{},
		StrmGenerationTask{},
	} {
		if !db.Db.Migrator().HasTable(table) {
			t.Fatalf("批量建表应创建 %s 表", GetTableName(table))
		}
	}
}

func TestBatchCreateTableCreatesDirectoryUploadProcessedFilesTable(t *testing.T) {
	testDb, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("打开测试数据库失败: %v", err)
	}
	db.Db = testDb

	if err := BatchCreateTable(); err != nil {
		t.Fatalf("批量建表失败: %v", err)
	}
	if !db.Db.Migrator().HasTable(DirectoryUploadProcessedFile{}) {
		t.Fatal("批量建表应创建 directory_upload_processed_files 表")
	}
}

func TestInitDBDoesNotCreateDefaultAdmin(t *testing.T) {
	gin.SetMode(gin.TestMode)
	testDb, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("打开测试数据库失败: %v", err)
	}
	db.Db = testDb
	helpers.AppLogger = &helpers.QLogger{
		Logger: log.New(io.Discard, "", 0),
	}

	InitDB()

	var count int64
	if err := db.Db.Model(&User{}).Count(&count).Error; err != nil {
		t.Fatalf("统计用户失败: %v", err)
	}
	if count != 0 {
		t.Fatalf("新库初始化后用户数量 = %d，期望 0", count)
	}
}

func createMigratorTestTable(t *testing.T) {
	t.Helper()
	if err := db.Db.Exec(`
		CREATE TABLE migrator (
			id integer primary key autoincrement,
			created_at integer,
			updated_at integer,
			version_code integer
		)
	`).Error; err != nil {
		t.Fatalf("创建迁移表失败: %v", err)
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
	createMigratorTestTable(t)
	if err := db.Db.AutoMigrate(&DbDownloadTask{}, &DbUploadTask{}, &Settings{}, &SyncPath{}); err != nil {
		t.Fatalf("创建测试表失败: %v", err)
	}
	if err := db.Db.Create(&Migrator{VersionCode: 43}).Error; err != nil {
		t.Fatalf("创建迁移版本记录失败: %v", err)
	}
}

type legacyUniqueNotificationChannel struct {
	ID          uint   `gorm:"primaryKey"`
	ChannelType string `gorm:"index,uniqueIndex:idx_channel_type"`
	ChannelName string
	IsEnabled   bool `gorm:"default:true"`
}

type legacyUserWithoutSingletonKey struct {
	BaseModel
	Username string `gorm:"unique;not null"`
	Password string `gorm:"not null"`
}

func (legacyUserWithoutSingletonKey) TableName() string {
	return "users"
}

func (legacyUniqueNotificationChannel) TableName() string {
	return "notification_channels"
}

func setupMigratorVersion45NotificationTestDB(t *testing.T) {
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
	createMigratorTestTable(t)
	if err := db.Db.AutoMigrate(&legacyUniqueNotificationChannel{}, &NotificationRule{}, &Settings{}, &SyncPath{}); err != nil {
		t.Fatalf("创建旧通知渠道测试表失败: %v", err)
	}
	if err := db.Db.Create(&Migrator{VersionCode: 45}).Error; err != nil {
		t.Fatalf("创建迁移版本记录失败: %v", err)
	}
	if err := db.Db.Create(&legacyUniqueNotificationChannel{ChannelType: "telegram", ChannelName: "Telegram A", IsEnabled: true}).Error; err != nil {
		t.Fatalf("创建旧通知渠道失败: %v", err)
	}
}

func TestMigrateNotificationChannelAllowsDuplicateTypesAndBackfillsRules(t *testing.T) {
	setupMigratorVersion45NotificationTestDB(t)

	Migrate()

	var migrator Migrator
	if err := db.Db.First(&migrator).Error; err != nil {
		t.Fatalf("读取迁移版本失败: %v", err)
	}
	if migrator.VersionCode != MaxVersionCode {
		t.Fatalf("迁移版本 = %d，期望 %d", migrator.VersionCode, MaxVersionCode)
	}

	if err := db.Db.Create(&NotificationChannel{ChannelType: "telegram", ChannelName: "Telegram B", IsEnabled: true}).Error; err != nil {
		t.Fatalf("迁移后应允许创建同类型通知渠道: %v", err)
	}

	var channel NotificationChannel
	if err := db.Db.Where("channel_name = ?", "Telegram A").First(&channel).Error; err != nil {
		t.Fatalf("读取已有通知渠道失败: %v", err)
	}
	var total int64
	if err := db.Db.Model(&NotificationRule{}).Where("channel_id = ?", channel.ID).Count(&total).Error; err != nil {
		t.Fatalf("统计通知规则失败: %v", err)
	}
	if total != int64(len(notification.AllNotificationTypes)) {
		t.Fatalf("补齐规则数量 = %d，期望 %d", total, len(notification.AllNotificationTypes))
	}
}

func TestMigrateVersion50AddsEmbyDailyFirstFullSyncFields(t *testing.T) {
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
	GlobalEmbyConfig = nil

	createMigratorTestTable(t)
	if err := db.Db.Create(&Migrator{VersionCode: 50}).Error; err != nil {
		t.Fatalf("创建迁移版本记录失败: %v", err)
	}
	if err := db.Db.Exec(`
		CREATE TABLE emby_config (
			id integer primary key autoincrement,
			created_at integer,
			updated_at integer,
			emby_url text,
			emby_api_key text,
			enable_delete_netdisk integer,
			enable_refresh_library integer,
			enable_media_notification integer,
			enable_extract_media_info integer,
			enable_auth integer,
			sync_enabled integer,
			sync_cron text,
			last_sync_time integer,
			last_full_sync_at integer,
			last_incremental_sync_at integer,
			last_saved_cursor_at integer,
			last_processed_count integer,
			last_error text,
			is_running numeric,
			sync_mode text,
			started_at integer,
			selected_libraries text,
			sync_all_libraries integer,
			enable_playback_overview integer,
			enable_playback_progress integer
		)
	`).Error; err != nil {
		t.Fatalf("创建版本 50 emby_config 表失败: %v", err)
	}
	if err := db.Db.Exec(`
		INSERT INTO emby_config (
			created_at, updated_at, emby_url, emby_api_key,
			enable_delete_netdisk, enable_refresh_library, enable_media_notification, enable_extract_media_info, enable_auth,
			sync_enabled, sync_cron,
			last_sync_time, last_full_sync_at, last_incremental_sync_at, last_saved_cursor_at,
			last_processed_count, last_error, is_running, sync_mode, started_at,
			selected_libraries, sync_all_libraries, enable_playback_overview, enable_playback_progress
		)
		VALUES (
			1, 1, 'http://emby.local', 'key',
			0, 1, 0, 1, 1,
			1, '0 * * * *',
			200, 200, 100, 90,
			12, '', 0, 'idle', 0,
			'[]', 1, 0, 0
		)
	`).Error; err != nil {
		t.Fatalf("插入版本 50 EmbyConfig 失败: %v", err)
	}

	Migrate()

	var migrator Migrator
	if err := db.Db.First(&migrator).Error; err != nil {
		t.Fatalf("读取迁移版本失败: %v", err)
	}
	if migrator.VersionCode != MaxVersionCode {
		t.Fatalf("迁移版本 = %d，期望 %d", migrator.VersionCode, MaxVersionCode)
	}
	for _, column := range []string{
		"enable_daily_first_full_sync",
		"last_success_sync_mode",
	} {
		if !db.Db.Migrator().HasColumn(&EmbyConfig{}, column) {
			t.Fatalf("迁移应添加 emby_config.%s 字段", column)
		}
	}

	var config EmbyConfig
	if err := db.Db.First(&config).Error; err != nil {
		t.Fatalf("读取 EmbyConfig 失败: %v", err)
	}
	if config.EnableDailyFirstFullSync != 1 {
		t.Fatalf("EnableDailyFirstFullSync = %d, want 1", config.EnableDailyFirstFullSync)
	}
	if config.LastSuccessSyncMode != EmbySyncModeFull {
		t.Fatalf("LastSuccessSyncMode = %q, want %q", config.LastSuccessSyncMode, EmbySyncModeFull)
	}
}

func TestMigrateVersion46AddsUserSingletonKey(t *testing.T) {
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
	createMigratorTestTable(t)
	if err := db.Db.AutoMigrate(&legacyUserWithoutSingletonKey{}, &Settings{}, &SyncPath{}); err != nil {
		t.Fatalf("创建旧用户测试表失败: %v", err)
	}
	if err := db.Db.Create(&Migrator{VersionCode: 46}).Error; err != nil {
		t.Fatalf("创建迁移版本记录失败: %v", err)
	}
	if err := db.Db.Create(&legacyUserWithoutSingletonKey{Username: "admin", Password: "hashed"}).Error; err != nil {
		t.Fatalf("创建旧用户失败: %v", err)
	}

	Migrate()

	var migrator Migrator
	if err := db.Db.First(&migrator).Error; err != nil {
		t.Fatalf("读取迁移版本失败: %v", err)
	}
	if migrator.VersionCode != MaxVersionCode {
		t.Fatalf("迁移版本 = %d，期望 %d", migrator.VersionCode, MaxVersionCode)
	}
	if !db.Db.Migrator().HasColumn(&User{}, "singleton_key") {
		t.Fatal("迁移应添加 users.singleton_key 字段")
	}
	if err := db.Db.Create(&User{Username: "other", Password: "hashed"}).Error; err == nil {
		t.Fatal("迁移后创建第二个用户 error = nil，期望被唯一约束拒绝")
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
	if migrator.VersionCode != MaxVersionCode {
		t.Fatalf("迁移版本 = %d，期望 %d", migrator.VersionCode, MaxVersionCode)
	}
	if !db.Db.Migrator().HasTable(UserSession{}) {
		t.Fatal("迁移应创建 user_sessions 表")
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

func TestMigrateVersion49AddsEmbySyncStateAndBatchFields(t *testing.T) {
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
	GlobalEmbyConfig = nil

	createMigratorTestTable(t)
	if err := db.Db.Create(&Migrator{VersionCode: 49}).Error; err != nil {
		t.Fatalf("创建迁移版本记录失败: %v", err)
	}
	if err := db.Db.Exec(`
		CREATE TABLE emby_config (
			id integer primary key autoincrement,
			created_at integer,
			updated_at integer,
			emby_url text,
			emby_api_key text,
			sync_enabled integer,
			sync_cron text,
			last_sync_time integer,
			selected_libraries text,
			sync_all_libraries integer
		)
	`).Error; err != nil {
		t.Fatalf("创建旧 emby_config 表失败: %v", err)
	}
	if err := db.Db.Exec(`
		INSERT INTO emby_config (created_at, updated_at, emby_url, emby_api_key, sync_enabled, sync_cron, last_sync_time, selected_libraries, sync_all_libraries)
		VALUES (1, 1, 'http://emby.local', 'key', 1, '0 * * * *', 123, '[]', 1)
	`).Error; err != nil {
		t.Fatalf("插入旧 EmbyConfig 失败: %v", err)
	}
	if err := db.Db.Exec(`
		CREATE TABLE emby_media_items (
			id integer primary key autoincrement,
			created_at integer,
			updated_at integer,
			item_id text,
			item_id_int integer,
			name text,
			type text,
			parent_id text,
			library_id text
		)
	`).Error; err != nil {
		t.Fatalf("创建旧 emby_media_items 表失败: %v", err)
	}

	Migrate()

	var migrator Migrator
	if err := db.Db.First(&migrator).Error; err != nil {
		t.Fatalf("读取迁移版本失败: %v", err)
	}
	if migrator.VersionCode != MaxVersionCode {
		t.Fatalf("迁移版本 = %d，期望 %d", migrator.VersionCode, MaxVersionCode)
	}
	for _, column := range []string{
		"last_full_sync_at",
		"last_incremental_sync_at",
		"last_saved_cursor_at",
		"last_processed_count",
		"last_error",
		"is_running",
		"sync_mode",
		"started_at",
	} {
		if !db.Db.Migrator().HasColumn(&EmbyConfig{}, column) {
			t.Fatalf("迁移应添加 emby_config.%s 字段", column)
		}
	}
	for _, column := range []string{"last_seen_sync_run", "last_seen_at"} {
		if !db.Db.Migrator().HasColumn(&EmbyMediaItem{}, column) {
			t.Fatalf("迁移应添加 emby_media_items.%s 字段", column)
		}
	}
}

func TestMigrateVersion51AddsUploadStrmModels(t *testing.T) {
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

	createMigratorTestTable(t)
	if err := db.Db.AutoMigrate(&DbUploadTask{}, &Settings{}); err != nil {
		t.Fatalf("创建版本 51 测试表失败: %v", err)
	}
	if err := db.Db.Create(&Migrator{VersionCode: 51}).Error; err != nil {
		t.Fatalf("创建迁移版本记录失败: %v", err)
	}

	Migrate()

	var migrator Migrator
	if err := db.Db.First(&migrator).Error; err != nil {
		t.Fatalf("读取迁移版本失败: %v", err)
	}
	if migrator.VersionCode != MaxVersionCode {
		t.Fatalf("迁移版本 = %d，期望 %d", migrator.VersionCode, MaxVersionCode)
	}
	for _, table := range []any{
		UploadSession{},
		DirectoryUploadRule{},
		StrmGenerationTask{},
	} {
		if !db.Db.Migrator().HasTable(table) {
			t.Fatalf("迁移应创建 %s 表", GetTableName(table))
		}
	}
	for _, column := range []string{
		"sync_path_id",
		"relative_path",
		"uploaded_bytes",
		"upload_result",
		"resume_state",
		"source_cleanup_status",
		"source_cleanup_error",
		"source_deleted_at",
	} {
		if !db.Db.Migrator().HasColumn(&DbUploadTask{}, column) {
			t.Fatalf("迁移应添加 db_upload_tasks.%s 字段", column)
		}
	}
	for _, column := range []string{
		"upload_rapid_wait_enabled",
		"upload_rapid_wait_timeout_seconds",
		"upload_rapid_wait_interval_seconds",
		"upload_rapid_wait_min_size",
		"upload_rapid_wait_force_size",
		"upload_rapid_wait_skip_upload",
	} {
		if !db.Db.Migrator().HasColumn(&Settings{}, column) {
			t.Fatalf("迁移应添加 settings.%s 字段", column)
		}
	}
}

func TestMigrateVersion52AddsEmbyTargetedRefreshFields(t *testing.T) {
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

	createMigratorTestTable(t)
	if err := db.Db.Create(&Migrator{VersionCode: 52}).Error; err != nil {
		t.Fatalf("创建迁移版本记录失败: %v", err)
	}
	if err := db.Db.Exec(`
		CREATE TABLE emby_library_refresh_tasks (
			id integer primary key autoincrement,
			created_at integer,
			updated_at integer,
			library_id text,
			library_name text,
			sync_path_ids_str text,
			status text,
			last_event_at integer,
			refresh_after_at integer,
			deadline_at integer,
			last_checked_at integer,
			last_refresh_at integer,
			error text
		)
	`).Error; err != nil {
		t.Fatalf("创建版本 52 Emby 刷新任务表失败: %v", err)
	}

	Migrate()

	var migrator Migrator
	if err := db.Db.First(&migrator).Error; err != nil {
		t.Fatalf("读取迁移版本失败: %v", err)
	}
	if migrator.VersionCode != MaxVersionCode {
		t.Fatalf("迁移版本 = %d，期望 %d", migrator.VersionCode, MaxVersionCode)
	}
	for _, column := range []string{
		"target_type",
		"item_ids_str",
		"item_recursive",
		"fallback_library_id",
		"fallback_library_name",
	} {
		if !db.Db.Migrator().HasColumn(&EmbyLibraryRefreshTask{}, column) {
			t.Fatalf("迁移应添加 emby_library_refresh_tasks.%s 字段", column)
		}
	}
}

func TestMigrateVersion56AddsDirectoryUploadProcessedFiles(t *testing.T) {
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

	createMigratorTestTable(t)
	if err := db.Db.Create(&Migrator{VersionCode: 56}).Error; err != nil {
		t.Fatalf("创建迁移版本记录失败: %v", err)
	}
	if err := db.Db.Exec(`
		CREATE TABLE db_upload_tasks (
			id integer primary key autoincrement,
			created_at integer,
			updated_at integer,
			source text,
			account_id integer,
			sync_path_id integer,
			source_type text,
			local_full_path text,
			remote_file_id text,
			remote_path_id text,
			file_name text,
			status integer,
			file_size integer
		)
	`).Error; err != nil {
		t.Fatalf("创建版本 56 上传任务表失败: %v", err)
	}

	Migrate()

	var migrator Migrator
	if err := db.Db.First(&migrator).Error; err != nil {
		t.Fatalf("读取迁移版本失败: %v", err)
	}
	if migrator.VersionCode != MaxVersionCode {
		t.Fatalf("迁移版本 = %d，期望 %d", migrator.VersionCode, MaxVersionCode)
	}
	if !db.Db.Migrator().HasTable(DirectoryUploadProcessedFile{}) {
		t.Fatal("迁移应创建 directory_upload_processed_files 表")
	}
	for _, column := range []string{
		"source_fingerprint",
		"local_mtime_ns",
	} {
		if !db.Db.Migrator().HasColumn(&DbUploadTask{}, column) {
			t.Fatalf("迁移应添加 db_upload_tasks.%s 字段", column)
		}
	}
}

func TestMigrateVersion57AddsDirectoryUploadEnabled(t *testing.T) {
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

	createMigratorTestTable(t)
	if err := db.Db.Create(&Migrator{VersionCode: 57}).Error; err != nil {
		t.Fatalf("创建迁移版本记录失败: %v", err)
	}
	if err := db.Db.Exec(`
		CREATE TABLE sync_paths (
			id integer primary key autoincrement,
			created_at integer,
			updated_at integer,
			base_cid text,
			local_path text,
			remote_path text,
			source_type text,
			account_id integer,
			enable_cron numeric
		)
	`).Error; err != nil {
		t.Fatalf("创建版本 57 同步目录表失败: %v", err)
	}
	if err := db.Db.Exec(`
		CREATE TABLE directory_upload_rules (
			id integer primary key autoincrement,
			sync_path_id integer,
			enabled numeric
		)
	`).Error; err != nil {
		t.Fatalf("创建版本 57 目录监控规则表失败: %v", err)
	}
	if err := db.Db.Exec(`
		INSERT INTO sync_paths (id, created_at, updated_at, base_cid, local_path, remote_path, source_type, account_id, enable_cron)
		VALUES
			(1, 0, 0, 'enabled-root', '/strm/enabled', '/remote/enabled', '115', 1, 0),
			(2, 0, 0, 'disabled-root', '/strm/disabled', '/remote/disabled', '115', 1, 0)
	`).Error; err != nil {
		t.Fatalf("写入版本 57 同步目录失败: %v", err)
	}
	if err := db.Db.Exec(`
		INSERT INTO directory_upload_rules (sync_path_id, enabled)
		VALUES (1, true), (2, false)
	`).Error; err != nil {
		t.Fatalf("写入版本 57 目录监控规则失败: %v", err)
	}

	Migrate()

	var migrator Migrator
	if err := db.Db.First(&migrator).Error; err != nil {
		t.Fatalf("读取迁移版本失败: %v", err)
	}
	if migrator.VersionCode != MaxVersionCode {
		t.Fatalf("迁移版本 = %d，期望 %d", migrator.VersionCode, MaxVersionCode)
	}
	if !db.Db.Migrator().HasColumn(&SyncPath{}, "directory_upload_enabled") {
		t.Fatal("迁移应添加 sync_paths.directory_upload_enabled 字段")
	}
	var enabledSyncPath SyncPath
	if err := db.Db.First(&enabledSyncPath, 1).Error; err != nil {
		t.Fatalf("读取启用规则同步目录失败: %v", err)
	}
	if !enabledSyncPath.DirectoryUploadEnabled {
		t.Fatal("存在启用目录监控规则的同步目录应回填总开关为 true")
	}
	var disabledSyncPath SyncPath
	if err := db.Db.First(&disabledSyncPath, 2).Error; err != nil {
		t.Fatalf("读取停用规则同步目录失败: %v", err)
	}
	if disabledSyncPath.DirectoryUploadEnabled {
		t.Fatal("只有停用目录监控规则的同步目录总开关应保持 false")
	}
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
