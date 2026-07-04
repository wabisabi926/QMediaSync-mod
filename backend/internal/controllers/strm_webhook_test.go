package controllers

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"qmediasync/internal/db"
	"qmediasync/internal/helpers"
	"qmediasync/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func setupStrmWebhookControllerTest(t *testing.T) (*gin.Engine, string, *models.SyncPath) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	helpers.AppLogger = &helpers.QLogger{Logger: log.New(io.Discard, "", 0)}
	testDB, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("打开测试数据库失败: %v", err)
	}
	db.Db = testDB
	if err := db.Db.AutoMigrate(
		&models.User{},
		&models.ApiKey{},
		&models.Account{},
		&models.SyncPath{},
		&models.StrmGenerationTask{},
	); err != nil {
		t.Fatalf("迁移测试表失败: %v", err)
	}
	user := &models.User{Username: "admin", Password: "hashed"}
	if err := db.Db.Create(user).Error; err != nil {
		t.Fatalf("创建用户失败: %v", err)
	}
	_, rawKey, err := models.CreateAPIKey(user.ID, "strm webhook")
	if err != nil {
		t.Fatalf("创建 API Key 失败: %v", err)
	}
	account := &models.Account{SourceType: models.SourceType115, Name: "115"}
	if err := db.Db.Create(account).Error; err != nil {
		t.Fatalf("创建账号失败: %v", err)
	}
	syncPath := &models.SyncPath{
		SourceType: models.SourceType115,
		AccountId:  account.ID,
		BaseCid:    "root",
		LocalPath:  t.TempDir(),
		RemotePath: "/remote",
	}
	if err := db.Db.Create(syncPath).Error; err != nil {
		t.Fatalf("创建同步目录失败: %v", err)
	}
	router := gin.New()
	router.POST("/api/strm/webhook", StrmWebhook)
	return router, rawKey, syncPath
}

func TestStrmWebhookAuthSupportsHeaderAndQueryAPIKey(t *testing.T) {
	router, rawKey, syncPath := setupStrmWebhookControllerTest(t)
	payload := map[string]any{
		"sync_path_id": syncPath.ID,
		"file_id":      "file-1",
		"pick_code":    "pick-1",
		"file_name":    "movie.mkv",
	}

	w := performStrmWebhookRequest(t, router, rawKey, "", payload)
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), `"accepted_count":1`) {
		t.Fatalf("header API Key 响应异常: code=%d body=%s", w.Code, w.Body.String())
	}

	w = performStrmWebhookRequest(t, router, "", rawKey, map[string]any{
		"sync_path_id": syncPath.ID,
		"file_id":      "file-2",
		"pick_code":    "pick-2",
		"file_name":    "movie2.mkv",
	})
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), `"accepted_count":1`) {
		t.Fatalf("query API Key 响应异常: code=%d body=%s", w.Code, w.Body.String())
	}

	w = performStrmWebhookRequest(t, router, "bad-key", "", payload)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("无效 API Key 状态码 = %d，期望 401，body=%s", w.Code, w.Body.String())
	}
}

func TestStrmWebhookValidatesFileLocator(t *testing.T) {
	router, rawKey, syncPath := setupStrmWebhookControllerTest(t)
	cases := []struct {
		name    string
		payload map[string]any
		want    string
	}{
		{
			name:    "缺少同步目录",
			payload: map[string]any{"file_id": "file-1"},
			want:    "sync_path_id",
		},
		{
			name:    "缺少文件定位",
			payload: map[string]any{"sync_path_id": syncPath.ID, "file_name": "movie.mkv"},
			want:    "file_id、pick_code 或 path + file_name",
		},
		{
			name: "拒绝 local_path",
			payload: map[string]any{
				"sync_path_id": syncPath.ID,
				"file_id":      "file-1",
				"local_path":   "/tmp/evil.strm",
			},
			want: "local_path",
		},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			w := performStrmWebhookRequest(t, router, rawKey, "", tt.payload)
			if w.Code != http.StatusBadRequest || !strings.Contains(w.Body.String(), tt.want) {
				t.Fatalf("响应异常: code=%d body=%s want=%s", w.Code, w.Body.String(), tt.want)
			}
		})
	}
}

func TestStrmWebhookEnqueuesFileTaskIdempotently(t *testing.T) {
	router, rawKey, syncPath := setupStrmWebhookControllerTest(t)
	payload := map[string]any{
		"sync_path_id": syncPath.ID,
		"file_id":      "file-1",
		"pick_code":    "pick-1",
		"parent_id":    "parent-1",
		"path":         "/remote",
		"file_name":    "movie.mkv",
		"file_size":    1024,
		"sha1":         "sha1",
		"mtime":        123,
	}
	w := performStrmWebhookRequest(t, router, rawKey, "", payload)
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), `"accepted_count":1`) {
		t.Fatalf("首次入队响应异常: code=%d body=%s", w.Code, w.Body.String())
	}
	w = performStrmWebhookRequest(t, router, rawKey, "", payload)
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), `"accepted_count":1`) {
		t.Fatalf("重复入队响应异常: code=%d body=%s", w.Code, w.Body.String())
	}
	var total int64
	if err := db.Db.Model(&models.StrmGenerationTask{}).Count(&total).Error; err != nil {
		t.Fatalf("统计 STRM 任务失败: %v", err)
	}
	if total != 1 {
		t.Fatalf("STRM 任务数量 = %d，期望 request_hash 去重为 1", total)
	}
	var task models.StrmGenerationTask
	if err := db.Db.First(&task).Error; err != nil {
		t.Fatalf("读取 STRM 任务失败: %v", err)
	}
	if task.Source != models.StrmGenerationSourceWebhook ||
		task.TaskType != models.StrmGenerationTaskTypeFile ||
		task.SyncPathId != syncPath.ID ||
		task.FileId != "file-1" ||
		task.PickCode != "pick-1" {
		t.Fatalf("STRM 任务 = %+v，期望 webhook file 任务", task)
	}
}

func TestStrmWebhookBatchFilesReturnsItemResults(t *testing.T) {
	router, rawKey, syncPath := setupStrmWebhookControllerTest(t)
	payload := map[string]any{
		"sync_path_id": syncPath.ID,
		"action":       "batch_files",
		"items": []map[string]any{
			{"file_id": "file-1", "pick_code": "pick-1", "file_name": "movie.mkv"},
			{"file_name": "missing.mkv"},
		},
	}
	w := performStrmWebhookRequest(t, router, rawKey, "", payload)
	body := w.Body.String()
	if w.Code != http.StatusOK ||
		!strings.Contains(body, `"accepted_count":1`) ||
		!strings.Contains(body, `"failed_count":1`) ||
		!strings.Contains(body, `"accepted":true`) ||
		!strings.Contains(body, `"accepted":false`) {
		t.Fatalf("批量响应异常: code=%d body=%s", w.Code, body)
	}
	var total int64
	if err := db.Db.Model(&models.StrmGenerationTask{}).Count(&total).Error; err != nil {
		t.Fatalf("统计 STRM 任务失败: %v", err)
	}
	if total != 1 {
		t.Fatalf("STRM 任务数量 = %d，期望只入队合法项", total)
	}
}

func TestStrmWebhookDirectoryScan(t *testing.T) {
	router, rawKey, syncPath := setupStrmWebhookControllerTest(t)
	w := performStrmWebhookRequest(t, router, rawKey, "", map[string]any{
		"sync_path_id":   syncPath.ID,
		"action":         "directory_scan",
		"directory_id":   "dir-1",
		"directory_path": "/remote/show",
	})
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), `"accepted_count":1`) {
		t.Fatalf("目录扫描响应异常: code=%d body=%s", w.Code, w.Body.String())
	}
	var task models.StrmGenerationTask
	if err := db.Db.First(&task).Error; err != nil {
		t.Fatalf("读取目录扫描任务失败: %v", err)
	}
	if task.TaskType != models.StrmGenerationTaskTypeDirectoryScan ||
		task.DirectoryId != "dir-1" ||
		task.DirectoryPath != "/remote/show" {
		t.Fatalf("目录扫描任务 = %+v，期望 directory_scan", task)
	}

	w = performStrmWebhookRequest(t, router, rawKey, "", map[string]any{
		"sync_path_id":   syncPath.ID,
		"action":         "directory_scan",
		"directory_path": "/other/show",
	})
	if w.Code != http.StatusBadRequest || !strings.Contains(w.Body.String(), "不在同步远端目录") {
		t.Fatalf("非法目录响应异常: code=%d body=%s", w.Code, w.Body.String())
	}
}

func performStrmWebhookRequest(t *testing.T, router *gin.Engine, headerKey string, queryKey string, payload map[string]any) *httptest.ResponseRecorder {
	t.Helper()
	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("构造请求失败: %v", err)
	}
	path := "/api/strm/webhook"
	if queryKey != "" {
		path += "?api_key=" + queryKey
	}
	req := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if headerKey != "" {
		req.Header.Set(apiKeyHeaderName, headerKey)
	}
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}
