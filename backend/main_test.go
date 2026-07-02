package main

import (
	"bytes"
	"log"
	"strings"
	"testing"

	"qmediasync/internal/controllers"
	"qmediasync/internal/db"
	"qmediasync/internal/helpers"
	"qmediasync/internal/models"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func setupInitialAdminLogTest(t *testing.T) *bytes.Buffer {
	t.Helper()

	testDb, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("打开测试数据库失败: %v", err)
	}
	db.Db = testDb
	if err := db.Db.AutoMigrate(&models.User{}); err != nil {
		t.Fatalf("迁移用户表失败: %v", err)
	}

	oldLogger := helpers.AppLogger
	oldLevel := helpers.ConfiguredLogLevel()
	var buf bytes.Buffer
	helpers.AppLogger = &helpers.QLogger{Logger: log.New(&buf, "", 0)}
	helpers.SetGlobalLogLevel(helpers.LogLevelError)
	t.Cleanup(func() {
		controllers.DisableInitialSetup()
		helpers.AppLogger = oldLogger
		helpers.SetGlobalLogLevel(oldLevel)
	})

	return &buf
}

func TestConfigureInitialAdminSetup在Error日志等级仍输出初始化码(t *testing.T) {
	buf := setupInitialAdminLogTest(t)

	if err := configureInitialAdminSetup(); err != nil {
		t.Fatalf("configureInitialAdminSetup() error = %v", err)
	}

	got := buf.String()
	linePrefix := "[WARN] 检测到系统尚未创建管理员，请使用以下初始化码完成首次管理员创建："
	idx := strings.Index(got, linePrefix)
	if idx < 0 {
		t.Fatalf("Error 日志等级下应仍输出初始化码 Warn 日志: %s", got)
	}
	tokenLine := got[idx+len(linePrefix):]
	token := strings.TrimSpace(strings.SplitN(tokenLine, "\n", 2)[0])
	if token == "" {
		t.Fatalf("初始化码日志缺少 token: %s", got)
	}
	if !strings.Contains(got, "[WARN] 初始化码只会在本次启动日志中显示，创建管理员成功后立即失效") {
		t.Fatalf("初始化码失效提示应随初始化码一起输出: %s", got)
	}
}
