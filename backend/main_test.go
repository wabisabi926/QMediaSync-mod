package main

import (
	"bytes"
	"log"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"qmediasync/internal/controllers"
	"qmediasync/internal/db"
	"qmediasync/internal/helpers"
	"qmediasync/internal/models"

	"github.com/gin-gonic/gin"
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

func TestSyncPathAggregateWriteRoutesReplaceLegacyRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	oldLogger := helpers.AppLogger
	helpers.AppLogger = &helpers.QLogger{Logger: log.New(&bytes.Buffer{}, "", 0)}
	t.Cleanup(func() { helpers.AppLogger = oldLogger })
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "web_statics"), 0o755); err != nil {
		t.Fatalf("创建测试静态目录失败: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "web_statics", "index.html"), []byte("<html></html>"), 0o644); err != nil {
		t.Fatalf("创建测试 index.html 失败: %v", err)
	}
	t.Chdir(root)
	router := gin.New()
	setRouter(router)

	routes := make(map[string]struct{})
	for _, route := range router.Routes() {
		routes[route.Method+" "+route.Path] = struct{}{}
	}
	for _, expected := range []string{"POST /api/sync/paths", "PUT /api/sync/paths/:id"} {
		if _, ok := routes[expected]; !ok {
			t.Fatalf("缺少新同步目录写路由 %s", expected)
		}
	}
	for _, removed := range []string{
		"POST /api/sync/path-add",
		"POST /api/sync/path-update",
	} {
		if _, ok := routes[removed]; ok {
			t.Fatalf("旧写路由仍存在：%s", removed)
		}
	}
	for _, retained := range []string{
		"GET /api/sync/path/:id",
	} {
		if _, ok := routes[retained]; !ok {
			t.Fatalf("应保留的查询或运行接口缺失：%s", retained)
		}
	}
}

func TestLegacySyncWriteHandlersRemovedFromControllerSources(t *testing.T) {
	files := []struct {
		path    string
		removed []string
	}{
		{
			path: "internal/controllers/sync.go",
			removed: []string{
				"func AddSyncPath(",
				"func UpdateSyncPath(",
				"@Router /sync/path-add [post]",
				"@Router /sync/path-update [post]",
			},
		},
	}
	for _, file := range files {
		source, err := os.ReadFile(file.path)
		if err != nil {
			t.Fatalf("读取 %s 失败: %v", file.path, err)
		}
		for _, removed := range file.removed {
			if strings.Contains(string(source), removed) {
				t.Fatalf("%s 仍包含旧写接口源码标记 %q", file.path, removed)
			}
		}
	}
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
