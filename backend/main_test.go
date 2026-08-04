package main

import (
	"bytes"
	"log"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"qmediasync/internal/helpers"

	"github.com/gin-gonic/gin"
)

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
		"GET /api/directory-upload/rules",
		"POST /api/directory-upload/sync-paths/:sync_path_id/scan",
		"GET /api/directory-upload/runtime-status",
	} {
		if _, ok := routes[retained]; !ok {
			t.Fatalf("应保留的查询或运行接口缺失：%s", retained)
		}
	}
}

func TestLegacySyncWriteHandlersRemovedFromControllerSources(t *testing.T) {
	files := []struct {
		path     string
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
		{
			path: "internal/controllers/directory_upload.go",
			removed: []string{
				"func SaveDirectoryUploadSyncPathRules(",
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
