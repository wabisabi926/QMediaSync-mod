package scrape

import (
	"io"
	"log"
	"testing"

	"qmediasync/internal/db"
	"qmediasync/internal/helpers"
	"qmediasync/internal/models"
	"qmediasync/internal/syncstrm"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func setupTemporarySyncRecordTestDB(t *testing.T) {
	t.Helper()

	originalDb := db.Db
	originalConfigDir := helpers.ConfigDir
	originalGlobalConfig := helpers.GlobalConfig
	originalLogger := helpers.AppLogger
	t.Cleanup(func() {
		db.Db = originalDb
		helpers.ConfigDir = originalConfigDir
		helpers.GlobalConfig = originalGlobalConfig
		helpers.AppLogger = originalLogger
	})

	testDb, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("打开测试数据库失败: %v", err)
	}
	db.Db = testDb
	if err := db.Db.AutoMigrate(&models.Sync{}); err != nil {
		t.Fatalf("迁移测试表失败: %v", err)
	}
	helpers.ConfigDir = t.TempDir()
	helpers.GlobalConfig = *helpers.MakeDefaultConfig()
	helpers.AppLogger = &helpers.QLogger{Logger: log.New(io.Discard, "", 0)}
}

func TestCleanupTemporarySyncRecord删除待处理临时记录(t *testing.T) {
	setupTemporarySyncRecordTestDB(t)
	sync := &models.Sync{
		SyncPathId: 1,
		Status:     models.SyncStatusPending,
		LocalPath:  "/local",
		RemotePath: "/remote",
	}
	if err := db.Db.Create(sync).Error; err != nil {
		t.Fatalf("创建临时同步记录失败: %v", err)
	}

	cleanupTemporarySyncRecord(&syncstrm.SyncStrm{Sync: sync})

	var count int64
	if err := db.Db.Model(&models.Sync{}).Where("id = ?", sync.ID).Count(&count).Error; err != nil {
		t.Fatalf("统计同步记录失败: %v", err)
	}
	if count != 0 {
		t.Fatalf("临时同步记录未清理，count = %d", count)
	}
}

func TestCleanupTemporarySyncRecord空值不崩溃(t *testing.T) {
	cleanupTemporarySyncRecord(nil)
	cleanupTemporarySyncRecord(&syncstrm.SyncStrm{})
}
