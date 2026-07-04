package controllers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"qmediasync/internal/db"
	"qmediasync/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func setupDirectoryUploadControllerTest(t *testing.T) (*gin.Engine, *models.SyncPath) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	testDB, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("打开测试数据库失败: %v", err)
	}
	db.Db = testDB
	if err := db.Db.AutoMigrate(
		&models.Account{},
		&models.SyncPath{},
		&models.DirectoryUploadRule{},
	); err != nil {
		t.Fatalf("迁移测试表失败: %v", err)
	}
	models.SettingsGlobal = &models.Settings{
		SettingStrm: models.SettingStrm{
			VideoExtArr:  []string{".mkv", ".mp4"},
			MetaExtArr:   []string{".nfo"},
			MinVideoSize: 0,
		},
	}
	account := &models.Account{SourceType: models.SourceType115, Name: "115"}
	if err := db.Db.Create(account).Error; err != nil {
		t.Fatalf("创建账号失败: %v", err)
	}
	syncPath := &models.SyncPath{
		SourceType:  models.SourceType115,
		AccountId:   account.ID,
		BaseCid:     "root",
		LocalPath:   filepath.Join(t.TempDir(), "strm"),
		RemotePath:  "/remote",
		SettingStrm: models.SettingStrm{VideoExtArr: []string{".mkv", ".mp4"}, MinVideoSize: 0},
	}
	if err := db.Db.Create(syncPath).Error; err != nil {
		t.Fatalf("创建同步目录失败: %v", err)
	}

	router := gin.New()
	router.GET("/rules", ListDirectoryUploadRules)
	router.POST("/rules", CreateDirectoryUploadRule)
	router.PUT("/rules/:id", UpdateDirectoryUploadRule)
	router.DELETE("/rules/:id", DeleteDirectoryUploadRule)
	router.POST("/rules/:id/status", SetDirectoryUploadRuleStatus)
	router.POST("/rules/:id/scan", ScanDirectoryUploadRule)
	return router, syncPath
}

func TestDirectoryUploadRuleCRUDAndStatus(t *testing.T) {
	router, syncPath := setupDirectoryUploadControllerTest(t)
	monitorPath := t.TempDir()
	payload := map[string]any{
		"sync_path_id":                syncPath.ID,
		"account_id":                  syncPath.AccountId,
		"enabled":                     true,
		"monitor_path":                monitorPath,
		"remote_root_path":            "/remote/uploads",
		"remote_root_id":              "remote-root",
		"recursive":                   false,
		"upload_metadata":             true,
		"startup_scan_enabled":        false,
		"delete_source_after_success": true,
	}
	body, _ := json.Marshal(payload)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/rules", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	createBody := w.Body.String()
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), `"code":200`) {
		t.Fatalf("创建规则响应异常: code=%d body=%s", w.Code, w.Body.String())
	}

	var rule models.DirectoryUploadRule
	if err := db.Db.First(&rule).Error; err != nil {
		t.Fatalf("读取目录上传规则失败: %v", err)
	}
	if rule.Recursive || !rule.UploadMetadata || rule.StartupScanEnabled || !rule.DeleteSourceAfterSuccess {
		t.Fatalf("规则布尔字段 = %+v，期望保留显式开关，创建响应: %s", rule, createBody)
	}

	updateBody, _ := json.Marshal(map[string]any{
		"sync_path_id":                syncPath.ID,
		"account_id":                  syncPath.AccountId,
		"enabled":                     true,
		"monitor_path":                monitorPath,
		"remote_root_path":            "/remote/uploads",
		"remote_root_id":              "remote-root",
		"recursive":                   true,
		"upload_metadata":             false,
		"startup_scan_enabled":        true,
		"delete_source_after_success": false,
	})
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPut, "/rules/1", bytes.NewReader(updateBody))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), `"code":200`) {
		t.Fatalf("更新规则响应异常: code=%d body=%s", w.Code, w.Body.String())
	}

	w = httptest.NewRecorder()
	statusBody, _ := json.Marshal(map[string]any{"enabled": false})
	req = httptest.NewRequest(http.MethodPost, "/rules/1/status", bytes.NewReader(statusBody))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), `"code":200`) {
		t.Fatalf("启停规则响应异常: code=%d body=%s", w.Code, w.Body.String())
	}
	if err := db.Db.First(&rule, 1).Error; err != nil {
		t.Fatalf("读取更新后规则失败: %v", err)
	}
	if rule.Enabled {
		t.Fatal("规则应被停用")
	}
	if !rule.Recursive || rule.UploadMetadata || !rule.StartupScanEnabled || rule.DeleteSourceAfterSuccess {
		t.Fatalf("更新后规则布尔字段 = %+v，期望保留更新值", rule)
	}

	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/rules?sync_path_id=1", nil)
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), `"total":1`) {
		t.Fatalf("列表响应异常: code=%d body=%s", w.Code, w.Body.String())
	}

	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodDelete, "/rules/1", nil)
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), `"code":200`) {
		t.Fatalf("删除规则响应异常: code=%d body=%s", w.Code, w.Body.String())
	}
}

func TestScanDirectoryUploadRuleReturnsAcceptedCount(t *testing.T) {
	router, syncPath := setupDirectoryUploadControllerTest(t)
	monitorPath := t.TempDir()
	if err := os.WriteFile(filepath.Join(monitorPath, "movie.mkv"), []byte("movie"), 0o644); err != nil {
		t.Fatalf("写入测试视频失败: %v", err)
	}
	if err := os.WriteFile(filepath.Join(monitorPath, "ignore.tmp"), []byte("tmp"), 0o644); err != nil {
		t.Fatalf("写入测试临时文件失败: %v", err)
	}
	if err := db.Db.Create(&models.DirectoryUploadRule{
		SyncPathId:                    syncPath.ID,
		AccountId:                     syncPath.AccountId,
		Enabled:                       true,
		MonitorPath:                   monitorPath,
		RemoteRootPath:                "/remote",
		RemoteRootId:                  "remote-root",
		Recursive:                     true,
		WatchMode:                     models.DirectoryUploadWatchModePolling,
		StabilitySeconds:              0,
		StabilityCheckIntervalSeconds: 1,
		StabilityRequiredCount:        1,
		ProcessedCacheTTLSeconds:      600,
	}).Error; err != nil {
		t.Fatalf("创建目录上传规则失败: %v", err)
	}

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/rules/1/scan", nil)
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), `"accepted":1`) {
		t.Fatalf("扫描响应异常: code=%d body=%s", w.Code, w.Body.String())
	}
}

func TestScanDirectoryUploadRuleRejectsDisabledRule(t *testing.T) {
	router, syncPath := setupDirectoryUploadControllerTest(t)
	monitorPath := t.TempDir()
	if err := os.WriteFile(filepath.Join(monitorPath, "movie.mkv"), []byte("movie"), 0o644); err != nil {
		t.Fatalf("写入测试视频失败: %v", err)
	}
	rule := &models.DirectoryUploadRule{
		SyncPathId:               syncPath.ID,
		AccountId:                syncPath.AccountId,
		Enabled:                  true,
		MonitorPath:              monitorPath,
		RemoteRootPath:           "/remote",
		RemoteRootId:             "remote-root",
		Recursive:                true,
		WatchMode:                models.DirectoryUploadWatchModePolling,
		ProcessedCacheTTLSeconds: 600,
	}
	if err := db.Db.Create(rule).Error; err != nil {
		t.Fatalf("创建目录上传规则失败: %v", err)
	}
	if err := models.SetDirectoryUploadRuleEnabled(rule.ID, false); err != nil {
		t.Fatalf("停用目录上传规则失败: %v", err)
	}

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/rules/%d/scan", rule.ID), nil)
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK ||
		!strings.Contains(w.Body.String(), `"code":500`) ||
		!strings.Contains(w.Body.String(), "未启用") {
		t.Fatalf("停用规则扫描响应异常: code=%d body=%s", w.Code, w.Body.String())
	}
}

func TestCreateDirectoryUploadRuleRejectsPathOutsideSyncRemote(t *testing.T) {
	router, syncPath := setupDirectoryUploadControllerTest(t)
	payload := map[string]any{
		"sync_path_id":     syncPath.ID,
		"account_id":       syncPath.AccountId,
		"monitor_path":     t.TempDir(),
		"remote_root_path": "/other",
		"remote_root_id":   "remote-root",
	}
	body, _ := json.Marshal(payload)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/rules", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest || !strings.Contains(w.Body.String(), "不在同步远端目录") {
		t.Fatalf("非法远端目录响应异常: code=%d body=%s", w.Code, w.Body.String())
	}
}

func TestDirectoryUploadControllerRejectsInvalidID(t *testing.T) {
	router, _ := setupDirectoryUploadControllerTest(t)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/rules/abc/scan", nil)
	router.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("非法 ID 状态码 = %d，期望 %d", w.Code, http.StatusBadRequest)
	}
}

func TestDirectoryUploadRuleDefaults(t *testing.T) {
	router, syncPath := setupDirectoryUploadControllerTest(t)
	payload := map[string]any{
		"sync_path_id":     syncPath.ID,
		"account_id":       syncPath.AccountId,
		"monitor_path":     t.TempDir(),
		"remote_root_path": "/remote",
		"remote_root_id":   "remote-root",
	}
	body, _ := json.Marshal(payload)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/rules", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("创建默认规则失败: code=%d body=%s", w.Code, w.Body.String())
	}
	var rule models.DirectoryUploadRule
	if err := db.Db.First(&rule).Error; err != nil {
		t.Fatalf("读取默认规则失败: %v", err)
	}
	if !rule.Enabled || !rule.Recursive || !rule.StartupScanEnabled ||
		rule.WatchMode != models.DirectoryUploadWatchModeAuto ||
		rule.StabilitySeconds != 15 || rule.RescanIntervalSeconds != 30 ||
		rule.ProcessedCacheTTLSeconds <= 0 {
		t.Fatalf("默认规则 = %+v，期望填充默认值", rule)
	}
	if rule.DeleteSourceAfterSuccess {
		t.Fatal("删除源文件开关默认应关闭")
	}
	if rule.UploadMetadata {
		t.Fatal("上传元数据开关默认应关闭")
	}
}

func TestDirectoryUploadRuleIgnoresTimingFieldsFromPayload(t *testing.T) {
	router, syncPath := setupDirectoryUploadControllerTest(t)
	payload := map[string]any{
		"sync_path_id":                     syncPath.ID,
		"account_id":                       syncPath.AccountId,
		"monitor_path":                     t.TempDir(),
		"remote_root_path":                 "/remote",
		"remote_root_id":                   "remote-root",
		"stability_seconds":                3600,
		"stability_check_interval_seconds": 99,
		"stability_required_count":         99,
		"rescan_interval_seconds":          999,
	}
	body, _ := json.Marshal(payload)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/rules", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("创建规则失败: code=%d body=%s", w.Code, w.Body.String())
	}

	var rule models.DirectoryUploadRule
	if err := db.Db.First(&rule).Error; err != nil {
		t.Fatalf("读取默认规则失败: %v", err)
	}
	if rule.StabilitySeconds != 15 ||
		rule.StabilityCheckIntervalSeconds != 2 ||
		rule.StabilityRequiredCount != 3 ||
		rule.RescanIntervalSeconds != 30 {
		t.Fatalf("规则计时字段 = %+v，期望忽略请求值并使用内置默认值", rule)
	}
}

func TestDirectoryUploadRuleAcceptsNewEnumValues(t *testing.T) {
	router, syncPath := setupDirectoryUploadControllerTest(t)
	payload := map[string]any{
		"sync_path_id":     syncPath.ID,
		"account_id":       syncPath.AccountId,
		"monitor_path":     t.TempDir(),
		"remote_root_path": "/remote",
		"remote_root_id":   "remote-root",
		"watch_mode":       "fsnotify",
		"overwrite_mode":   "replace_conflict",
	}
	body, _ := json.Marshal(payload)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/rules", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("创建新枚举规则失败: code=%d body=%s", w.Code, w.Body.String())
	}
	var rule models.DirectoryUploadRule
	if err := db.Db.First(&rule).Error; err != nil {
		t.Fatalf("读取新枚举规则失败: %v", err)
	}
	if rule.WatchMode != models.DirectoryUploadWatchModeFSNotify ||
		rule.OverwriteMode != models.DirectoryUploadOverwriteReplaceConflict {
		t.Fatalf("规则枚举 = %+v，期望 fsnotify / replace_conflict", rule)
	}
}

func TestDirectoryUploadRuleRejectsLegacyEnumValues(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(map[string]any)
	}{
		{
			name: "拒绝旧 watcher 监控模式",
			mutate: func(payload map[string]any) {
				payload["watch_mode"] = "watcher"
			},
		},
		{
			name: "拒绝旧 always 覆盖策略",
			mutate: func(payload map[string]any) {
				payload["overwrite_mode"] = "always"
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router, syncPath := setupDirectoryUploadControllerTest(t)
			payload := map[string]any{
				"sync_path_id":     syncPath.ID,
				"account_id":       syncPath.AccountId,
				"monitor_path":     t.TempDir(),
				"remote_root_path": "/remote",
				"remote_root_id":   "remote-root",
			}
			tt.mutate(payload)
			body, _ := json.Marshal(payload)
			w := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, "/rules", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			router.ServeHTTP(w, req)
			if w.Code != http.StatusBadRequest || !strings.Contains(w.Body.String(), "不支持") {
				t.Fatalf("旧枚举响应异常: code=%d body=%s", w.Code, w.Body.String())
			}
		})
	}
}

func TestScanDirectoryUploadRuleWithMissingRule(t *testing.T) {
	router, _ := setupDirectoryUploadControllerTest(t)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/rules/999/scan", nil)
	router.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("缺失规则状态码 = %d，期望 %d", w.Code, http.StatusNotFound)
	}
}

func TestDirectoryUploadRuleScanIgnoresFutureMtime(t *testing.T) {
	router, syncPath := setupDirectoryUploadControllerTest(t)
	monitorPath := t.TempDir()
	filePath := filepath.Join(monitorPath, "future.mkv")
	if err := os.WriteFile(filePath, []byte("movie"), 0o644); err != nil {
		t.Fatalf("写入测试视频失败: %v", err)
	}
	future := time.Now().Add(time.Hour)
	if err := os.Chtimes(filePath, future, future); err != nil {
		t.Fatalf("设置测试视频 mtime 失败: %v", err)
	}
	if err := db.Db.Create(&models.DirectoryUploadRule{
		SyncPathId:                    syncPath.ID,
		AccountId:                     syncPath.AccountId,
		Enabled:                       true,
		MonitorPath:                   monitorPath,
		RemoteRootPath:                "/remote",
		RemoteRootId:                  "remote-root",
		Recursive:                     true,
		WatchMode:                     models.DirectoryUploadWatchModePolling,
		StabilitySeconds:              0,
		StabilityCheckIntervalSeconds: 1,
		StabilityRequiredCount:        1,
		ProcessedCacheTTLSeconds:      600,
	}).Error; err != nil {
		t.Fatalf("创建目录上传规则失败: %v", err)
	}
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/rules/1/scan", nil)
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), `"accepted":1`) {
		t.Fatalf("扫描响应异常: code=%d body=%s", w.Code, w.Body.String())
	}
}
