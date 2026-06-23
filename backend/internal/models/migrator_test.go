package models

import (
	"Q115-STRM/internal/db"
	"testing"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
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
