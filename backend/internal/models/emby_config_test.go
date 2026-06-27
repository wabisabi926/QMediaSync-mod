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
