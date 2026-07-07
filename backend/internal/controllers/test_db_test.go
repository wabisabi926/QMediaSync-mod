package controllers

import (
	"testing"

	"qmediasync/internal/db"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func setupControllerTestDB(t *testing.T, models ...any) *gorm.DB {
	t.Helper()

	testDB, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("打开测试数据库失败: %v", err)
	}
	sqlDB, err := testDB.DB()
	if err != nil {
		t.Fatalf("读取底层测试数据库失败: %v", err)
	}
	sqlDB.SetMaxOpenConns(1)
	sqlDB.SetMaxIdleConns(1)

	originalDB := db.Db
	originalRunner := runAuthBackgroundTask
	db.Db = testDB
	runAuthBackgroundTask = func(fn func()) {
		fn()
	}
	t.Cleanup(func() {
		db.Db = originalDB
		runAuthBackgroundTask = originalRunner
	})

	if len(models) > 0 {
		if err := db.Db.AutoMigrate(models...); err != nil {
			t.Fatalf("迁移测试表失败: %v", err)
		}
	}
	return testDB
}
