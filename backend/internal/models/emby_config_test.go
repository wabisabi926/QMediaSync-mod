package models

import (
	"io"
	"log"
	"testing"

	"qmediasync/internal/db"
	"qmediasync/internal/helpers"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestInitEmbyConfig默认开启Webhook鉴权(t *testing.T) {
	helpers.AppLogger = &helpers.QLogger{Logger: log.New(io.Discard, "", 0)}
	testDb, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("打开测试数据库失败: %v", err)
	}
	db.Db = testDb
	GlobalEmbyConfig = nil

	if err := db.Db.AutoMigrate(&EmbyConfig{}); err != nil {
		t.Fatalf("迁移 EmbyConfig 失败: %v", err)
	}

	InitEmbyConfig()

	var config EmbyConfig
	if err := db.Db.First(&config).Error; err != nil {
		t.Fatalf("查询 EmbyConfig 失败: %v", err)
	}
	if config.EnableAuth != 1 {
		t.Fatalf("EnableAuth = %d, want 1", config.EnableAuth)
	}
}

func TestEmbyConfig状态字段迁移默认值(t *testing.T) {
	helpers.AppLogger = &helpers.QLogger{Logger: log.New(io.Discard, "", 0)}
	testDb, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("打开测试数据库失败: %v", err)
	}
	db.Db = testDb
	GlobalEmbyConfig = nil

	if err := db.Db.Exec(`
		CREATE TABLE emby_config (
			id integer primary key autoincrement,
			created_at integer,
			updated_at integer,
			emby_url text,
			emby_api_key text,
			sync_enabled integer,
			sync_cron text,
			last_sync_time integer
		)
	`).Error; err != nil {
		t.Fatalf("创建旧 emby_config 表失败: %v", err)
	}
	if err := db.Db.Exec(`
		INSERT INTO emby_config (created_at, updated_at, emby_url, emby_api_key, sync_enabled, sync_cron, last_sync_time)
		VALUES (1, 1, 'http://emby.local', 'key', 1, '0 * * * *', 123)
	`).Error; err != nil {
		t.Fatalf("插入旧 EmbyConfig 失败: %v", err)
	}

	if err := db.Db.AutoMigrate(&EmbyConfig{}); err != nil {
		t.Fatalf("迁移 EmbyConfig 失败: %v", err)
	}

	var config EmbyConfig
	if err := db.Db.First(&config).Error; err != nil {
		t.Fatalf("查询 EmbyConfig 失败: %v", err)
	}
	if config.IsRunning {
		t.Fatal("IsRunning = true, want false")
	}
	if config.SyncMode != "" && config.SyncMode != EmbySyncModeIdle {
		t.Fatalf("SyncMode = %q, want empty or idle", config.SyncMode)
	}
	if config.StartedAt != 0 || config.LastFullSyncAt != 0 || config.LastIncrementalSyncAt != 0 || config.LastSavedCursorAt != 0 {
		t.Fatalf("状态时间字段默认值异常: %+v", config)
	}
	if config.LastProcessedCount != 0 || config.LastError != "" {
		t.Fatalf("状态结果字段默认值异常: %+v", config)
	}
}

func TestEmbySyncRunStateLifecycle(t *testing.T) {
	helpers.AppLogger = &helpers.QLogger{Logger: log.New(io.Discard, "", 0)}
	testDb, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("打开测试数据库失败: %v", err)
	}
	db.Db = testDb
	GlobalEmbyConfig = nil

	if err := db.Db.AutoMigrate(&EmbyConfig{}); err != nil {
		t.Fatalf("迁移 EmbyConfig 失败: %v", err)
	}
	config := &EmbyConfig{SyncEnabled: 1, SyncCron: "0 * * * *"}
	if err := db.Db.Create(config).Error; err != nil {
		t.Fatalf("创建 EmbyConfig 失败: %v", err)
	}

	started, err := StartEmbySyncRun(EmbySyncModeFull, 100)
	if err != nil {
		t.Fatalf("StartEmbySyncRun() error = %v", err)
	}
	if !started {
		t.Fatal("StartEmbySyncRun() started = false, want true")
	}

	started, err = StartEmbySyncRun(EmbySyncModeIncremental, 101)
	if err != nil {
		t.Fatalf("StartEmbySyncRun() second error = %v", err)
	}
	if started {
		t.Fatal("StartEmbySyncRun() second started = true, want false")
	}

	fresh, err := GetEmbyConfigFromDB()
	if err != nil {
		t.Fatalf("GetEmbyConfigFromDB() error = %v", err)
	}
	if !fresh.IsRunning || fresh.SyncMode != EmbySyncModeFull || fresh.StartedAt != 100 {
		t.Fatalf("运行中状态异常: %+v", fresh)
	}

	if err := FinishEmbySyncRun(EmbySyncModeFull, 42, 200, nil); err != nil {
		t.Fatalf("FinishEmbySyncRun() success error = %v", err)
	}
	fresh, err = GetEmbyConfigFromDB()
	if err != nil {
		t.Fatalf("GetEmbyConfigFromDB() after success error = %v", err)
	}
	if fresh.IsRunning || fresh.SyncMode != EmbySyncModeIdle || fresh.StartedAt != 0 {
		t.Fatalf("完成后运行状态异常: %+v", fresh)
	}
	if fresh.LastSyncTime != 200 || fresh.LastFullSyncAt != 200 || fresh.LastProcessedCount != 42 || fresh.LastError != "" {
		t.Fatalf("完成后结果状态异常: %+v", fresh)
	}
}

func TestEmbySyncRunFailureState(t *testing.T) {
	helpers.AppLogger = &helpers.QLogger{Logger: log.New(io.Discard, "", 0)}
	testDb, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("打开测试数据库失败: %v", err)
	}
	db.Db = testDb
	GlobalEmbyConfig = nil

	if err := db.Db.AutoMigrate(&EmbyConfig{}); err != nil {
		t.Fatalf("迁移 EmbyConfig 失败: %v", err)
	}
	if err := db.Db.Create(&EmbyConfig{SyncEnabled: 1, SyncCron: "0 * * * *"}).Error; err != nil {
		t.Fatalf("创建 EmbyConfig 失败: %v", err)
	}
	if started, err := StartEmbySyncRun(EmbySyncModeIncremental, 300); err != nil || !started {
		t.Fatalf("StartEmbySyncRun() = %v, %v, want true, nil", started, err)
	}

	if err := FinishEmbySyncRun(EmbySyncModeIncremental, 7, 400, errTestEmbySyncFailure{}); err != nil {
		t.Fatalf("FinishEmbySyncRun() failure error = %v", err)
	}

	fresh, err := GetEmbyConfigFromDB()
	if err != nil {
		t.Fatalf("GetEmbyConfigFromDB() error = %v", err)
	}
	if fresh.IsRunning || fresh.SyncMode != EmbySyncModeIdle || fresh.StartedAt != 0 {
		t.Fatalf("失败后运行状态异常: %+v", fresh)
	}
	if fresh.LastSyncTime != 0 || fresh.LastIncrementalSyncAt != 0 {
		t.Fatalf("失败后不应推进同步时间: %+v", fresh)
	}
	if fresh.LastError != "同步失败" || fresh.LastProcessedCount != 7 {
		t.Fatalf("失败后错误状态异常: %+v", fresh)
	}
}

type errTestEmbySyncFailure struct{}

func (errTestEmbySyncFailure) Error() string {
	return "同步失败"
}
