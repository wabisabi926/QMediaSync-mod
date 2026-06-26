package main

import (
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"qmediasync/internal/helpers"

	"github.com/gin-gonic/gin"
)

func setupRouterAuthTest(t *testing.T) *gin.Engine {
	t.Helper()

	gin.SetMode(gin.TestMode)
	rootDir := t.TempDir()
	webStaticsDir := filepath.Join(rootDir, "web_statics")
	if err := os.MkdirAll(webStaticsDir, 0755); err != nil {
		t.Fatalf("创建测试静态目录失败: %v", err)
	}
	if err := os.WriteFile(filepath.Join(webStaticsDir, "index.html"), []byte("<html></html>"), 0644); err != nil {
		t.Fatalf("创建测试首页失败: %v", err)
	}
	helpers.RootDir = rootDir
	helpers.AppLogger = &helpers.QLogger{Logger: log.New(io.Discard, "", 0)}

	r := gin.New()
	r.Use(gin.RecoveryWithWriter(io.Discard))
	setRouter(r)
	return r
}

func Test敏感APIRoutes未认证时返回401(t *testing.T) {
	r := setupRouterAuthTest(t)

	cases := []struct {
		name   string
		method string
		path   string
	}{
		{name: "日志 WebSocket", method: http.MethodGet, path: "/api/logs/ws"},
		{name: "旧日志读取", method: http.MethodGet, path: "/api/logs/old"},
		{name: "日志下载", method: http.MethodGet, path: "/api/logs/download"},
		{name: "事件 WebSocket", method: http.MethodGet, path: "/api/events/ws"},
		{name: "刮削记录导出", method: http.MethodGet, path: "/api/scrape/records/export"},
		{name: "刮削临时图片", method: http.MethodGet, path: "/api/scrape/tmp-image"},
		{name: "飞牛环境查询", method: http.MethodGet, path: "/api/path/is-fn-os"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, tc.path, nil)
			w := httptest.NewRecorder()

			r.ServeHTTP(w, req)

			if w.Code != http.StatusUnauthorized {
				t.Fatalf("HTTP = %d, want %d, body=%s", w.Code, http.StatusUnauthorized, w.Body.String())
			}
		})
	}
}

func Test飞牛访问路径更新无需认证(t *testing.T) {
	r := setupRouterAuthTest(t)

	originalAccessiblePathes := helpers.AccessiblePathes
	defer func() {
		helpers.AccessiblePathes = originalAccessiblePathes
	}()
	helpers.AccessiblePathes = ""

	req := httptest.NewRequest(http.MethodPost, "/api/update-fn-access-path", strings.NewReader("path=/vol1/media:/etc:/home/user"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("HTTP = %d, want %d, body=%s", w.Code, http.StatusOK, w.Body.String())
	}
	if helpers.AccessiblePathes != "/vol1/media:/home/user" {
		t.Fatalf("AccessiblePathes = %q, want %q", helpers.AccessiblePathes, "/vol1/media:/home/user")
	}
}
