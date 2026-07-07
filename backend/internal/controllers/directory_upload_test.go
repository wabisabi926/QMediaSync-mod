package controllers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"qmediasync/internal/db"
	"qmediasync/internal/directoryupload"
	"qmediasync/internal/models"

	"github.com/gin-gonic/gin"
)

func setupDirectoryUploadControllerTest(t *testing.T) (*gin.Engine, *models.SyncPath) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	setupControllerTestDB(
		t,
		&models.Account{},
		&models.SyncPath{},
		&models.DirectoryUploadRule{},
		&models.DirectoryUploadProcessedFile{},
	)
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
		SourceType:             models.SourceType115,
		AccountId:              account.ID,
		BaseCid:                "root",
		LocalPath:              filepath.Join(t.TempDir(), "strm"),
		RemotePath:             "/remote",
		DirectoryUploadEnabled: true,
		SettingStrm:            models.SettingStrm{VideoExtArr: []string{".mkv", ".mp4"}, MinVideoSize: 0},
	}
	if err := db.Db.Create(syncPath).Error; err != nil {
		t.Fatalf("创建同步目录失败: %v", err)
	}

	router := gin.New()
	router.GET("/rules", ListDirectoryUploadRules)
	router.PUT("/sync-paths/:sync_path_id/rules", SaveDirectoryUploadSyncPathRules)
	router.POST("/sync-paths/:sync_path_id/scan", ScanDirectoryUploadSyncPathRules)
	router.GET("/runtime-status", GetDirectoryUploadRuntimeStatuses)
	return router, syncPath
}

func TestSaveDirectoryUploadSyncPathRulesCreatesUpdatesListsAndDeletes(t *testing.T) {
	router, syncPath := setupDirectoryUploadControllerTest(t)
	monitorPath := t.TempDir()
	body, _ := json.Marshal(map[string]any{
		"enabled": true,
		"rules": []map[string]any{
			{
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
			},
		},
	})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/sync-paths/%d/rules", syncPath.ID), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	createBody := w.Body.String()
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), `"code":200`) {
		t.Fatalf("保存新规则响应异常: code=%d body=%s", w.Code, w.Body.String())
	}

	var rule models.DirectoryUploadRule
	if err := db.Db.First(&rule).Error; err != nil {
		t.Fatalf("读取目录上传规则失败: %v", err)
	}
	if rule.Recursive || !rule.UploadMetadata || rule.StartupScanEnabled || !rule.DeleteSourceAfterSuccess {
		t.Fatalf("规则布尔字段 = %+v，期望保留显式开关，创建响应: %s", rule, createBody)
	}

	updateBody, _ := json.Marshal(map[string]any{
		"enabled": false,
		"rules": []map[string]any{
			{
				"id":                          rule.ID,
				"sync_path_id":                syncPath.ID,
				"account_id":                  syncPath.AccountId,
				"enabled":                     false,
				"monitor_path":                monitorPath,
				"remote_root_path":            "/remote/uploads",
				"remote_root_id":              "remote-root",
				"recursive":                   true,
				"upload_metadata":             false,
				"startup_scan_enabled":        true,
				"delete_source_after_success": false,
			},
		},
	})
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPut, fmt.Sprintf("/sync-paths/%d/rules", syncPath.ID), bytes.NewReader(updateBody))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), `"code":200`) {
		t.Fatalf("更新规则响应异常: code=%d body=%s", w.Code, w.Body.String())
	}
	if err := db.Db.First(&rule, rule.ID).Error; err != nil {
		t.Fatalf("读取更新后规则失败: %v", err)
	}
	if rule.Enabled {
		t.Fatal("规则应被停用")
	}
	if !rule.Recursive || rule.UploadMetadata || !rule.StartupScanEnabled || rule.DeleteSourceAfterSuccess {
		t.Fatalf("更新后规则布尔字段 = %+v，期望保留更新值", rule)
	}

	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, fmt.Sprintf("/rules?sync_path_id=%d", syncPath.ID), nil)
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), `"total":1`) {
		t.Fatalf("列表响应异常: code=%d body=%s", w.Code, w.Body.String())
	}

	deleteBody, _ := json.Marshal(map[string]any{"enabled": false, "rules": []map[string]any{}})
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPut, fmt.Sprintf("/sync-paths/%d/rules", syncPath.ID), bytes.NewReader(deleteBody))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), `"code":200`) {
		t.Fatalf("删除规则响应异常: code=%d body=%s", w.Code, w.Body.String())
	}
	var total int64
	if err := db.Db.Model(&models.DirectoryUploadRule{}).Where("sync_path_id = ?", syncPath.ID).Count(&total).Error; err != nil {
		t.Fatalf("统计删除后规则失败: %v", err)
	}
	if total != 0 {
		t.Fatalf("删除后规则数量 = %d，期望 0", total)
	}
}

func TestListDirectoryUploadRulesReturnsIgnorePatterns(t *testing.T) {
	router, syncPath := setupDirectoryUploadControllerTest(t)
	rule := &models.DirectoryUploadRule{
		SyncPathId:                    syncPath.ID,
		AccountId:                     syncPath.AccountId,
		Enabled:                       true,
		MonitorPath:                   t.TempDir(),
		RemoteRootPath:                "/remote/uploads",
		RemoteRootId:                  "remote-root",
		Recursive:                     true,
		WatchMode:                     models.DirectoryUploadWatchModeAuto,
		StabilitySeconds:              models.DirectoryUploadDefaultStabilitySeconds,
		StabilityCheckIntervalSeconds: models.DirectoryUploadDefaultStabilityCheckIntervalSeconds,
		StabilityRequiredCount:        models.DirectoryUploadDefaultStabilityRequiredCount,
		RescanIntervalSeconds:         models.DirectoryUploadDefaultRescanIntervalSeconds,
		StartupScanEnabled:            true,
		ProcessedCacheTTLSeconds:      600,
		IgnorePatternsStr:             `["**/sample/**","*.tmp"]`,
		OverwriteMode:                 models.DirectoryUploadOverwriteSkipSame,
	}
	if err := db.Db.Create(rule).Error; err != nil {
		t.Fatalf("创建目录上传规则失败: %v", err)
	}

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/rules?sync_path_id=%d", syncPath.ID), nil)
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("查询规则响应状态异常: code=%d body=%s", w.Code, w.Body.String())
	}

	var response struct {
		Code int `json:"code"`
		Data struct {
			List []struct {
				IgnorePatterns []string `json:"ignore_patterns"`
			} `json:"list"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("解析规则列表响应失败: %v, body=%s", err, w.Body.String())
	}
	if response.Code != int(Success) || len(response.Data.List) != 1 {
		t.Fatalf("规则列表响应 = %+v，期望返回 1 条成功数据", response)
	}
	want := []string{"**/sample/**", "*.tmp"}
	if !reflect.DeepEqual(response.Data.List[0].IgnorePatterns, want) {
		t.Fatalf("ignore_patterns=%v，期望 %v", response.Data.List[0].IgnorePatterns, want)
	}
}

func TestScanDirectoryUploadSyncPathRuleReturnsAcceptedCount(t *testing.T) {
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
	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/sync-paths/%d/scan", syncPath.ID), nil)
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), `"accepted":1`) {
		t.Fatalf("扫描响应异常: code=%d body=%s", w.Code, w.Body.String())
	}
}

func TestScanDirectoryUploadSyncPathRulesScansAllEnabledRules(t *testing.T) {
	router, syncPath := setupDirectoryUploadControllerTest(t)
	monitorOne := t.TempDir()
	monitorTwo := t.TempDir()
	monitorDisabled := t.TempDir()
	if err := os.WriteFile(filepath.Join(monitorOne, "movie-one.mkv"), []byte("movie"), 0o644); err != nil {
		t.Fatalf("写入第一个测试视频失败: %v", err)
	}
	if err := os.WriteFile(filepath.Join(monitorTwo, "movie-two.mkv"), []byte("movie"), 0o644); err != nil {
		t.Fatalf("写入第二个测试视频失败: %v", err)
	}
	if err := os.WriteFile(filepath.Join(monitorDisabled, "disabled.mkv"), []byte("movie"), 0o644); err != nil {
		t.Fatalf("写入停用规则测试视频失败: %v", err)
	}
	rules := []*models.DirectoryUploadRule{
		{
			SyncPathId:               syncPath.ID,
			AccountId:                syncPath.AccountId,
			Enabled:                  true,
			MonitorPath:              monitorOne,
			RemoteRootPath:           "/remote/one",
			RemoteRootId:             "remote-one",
			Recursive:                true,
			WatchMode:                models.DirectoryUploadWatchModePolling,
			ProcessedCacheTTLSeconds: 600,
		},
		{
			SyncPathId:               syncPath.ID,
			AccountId:                syncPath.AccountId,
			Enabled:                  true,
			MonitorPath:              monitorTwo,
			RemoteRootPath:           "/remote/two",
			RemoteRootId:             "remote-two",
			Recursive:                true,
			WatchMode:                models.DirectoryUploadWatchModePolling,
			ProcessedCacheTTLSeconds: 600,
		},
		{
			SyncPathId:               syncPath.ID,
			AccountId:                syncPath.AccountId,
			Enabled:                  false,
			MonitorPath:              monitorDisabled,
			RemoteRootPath:           "/remote/disabled",
			RemoteRootId:             "remote-disabled",
			Recursive:                true,
			WatchMode:                models.DirectoryUploadWatchModePolling,
			ProcessedCacheTTLSeconds: 600,
		},
	}
	if err := db.Db.Create(&rules).Error; err != nil {
		t.Fatalf("创建目录上传规则失败: %v", err)
	}
	if err := db.Db.Model(rules[2]).Update("enabled", false).Error; err != nil {
		t.Fatalf("停用第三条目录上传规则失败: %v", err)
	}

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/sync-paths/%d/scan", syncPath.ID), nil)
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("批量扫描状态异常: code=%d body=%s", w.Code, w.Body.String())
	}

	var response struct {
		Code int `json:"code"`
		Data struct {
			Accepted int `json:"accepted"`
			Items    []struct {
				RuleID   uint `json:"rule_id"`
				Accepted int  `json:"accepted"`
			} `json:"items"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("解析批量扫描响应失败: %v, body=%s", err, w.Body.String())
	}
	if response.Code != int(Success) || response.Data.Accepted != 2 || len(response.Data.Items) != 2 {
		t.Fatalf("批量扫描响应 = %+v，期望只扫描 2 条启用规则并加入 2 个候选文件", response)
	}
}

func TestSaveDirectoryUploadSyncPathRulesDisablesMasterWithoutChangingRules(t *testing.T) {
	router, syncPath := setupDirectoryUploadControllerTest(t)
	otherSyncPath := &models.SyncPath{
		SourceType:             models.SourceType115,
		AccountId:              syncPath.AccountId,
		BaseCid:                "other-root",
		LocalPath:              filepath.Join(t.TempDir(), "other-strm"),
		RemotePath:             "/other",
		DirectoryUploadEnabled: true,
	}
	if err := db.Db.Create(otherSyncPath).Error; err != nil {
		t.Fatalf("创建其他同步目录失败: %v", err)
	}
	rules := []*models.DirectoryUploadRule{
		{
			SyncPathId:     syncPath.ID,
			AccountId:      syncPath.AccountId,
			Enabled:        true,
			MonitorPath:    filepath.Join(t.TempDir(), "one"),
			RemoteRootPath: "/remote/one",
			RemoteRootId:   "remote-one",
		},
		{
			SyncPathId:     syncPath.ID,
			AccountId:      syncPath.AccountId,
			Enabled:        true,
			MonitorPath:    filepath.Join(t.TempDir(), "two"),
			RemoteRootPath: "/remote/two",
			RemoteRootId:   "remote-two",
		},
		{
			SyncPathId:     otherSyncPath.ID,
			AccountId:      syncPath.AccountId,
			Enabled:        true,
			MonitorPath:    filepath.Join(t.TempDir(), "other"),
			RemoteRootPath: "/other/uploads",
			RemoteRootId:   "other-upload",
		},
	}
	if err := db.Db.Create(&rules).Error; err != nil {
		t.Fatalf("创建目录上传规则失败: %v", err)
	}

	body, _ := json.Marshal(map[string]any{
		"enabled": false,
		"rules": []map[string]any{
			{
				"id":               rules[0].ID,
				"sync_path_id":     syncPath.ID,
				"account_id":       syncPath.AccountId,
				"enabled":          true,
				"monitor_path":     rules[0].MonitorPath,
				"remote_root_path": rules[0].RemoteRootPath,
				"remote_root_id":   rules[0].RemoteRootId,
				"recursive":        true,
			},
			{
				"id":               rules[1].ID,
				"sync_path_id":     syncPath.ID,
				"account_id":       syncPath.AccountId,
				"enabled":          true,
				"monitor_path":     rules[1].MonitorPath,
				"remote_root_path": rules[1].RemoteRootPath,
				"remote_root_id":   rules[1].RemoteRootId,
				"recursive":        true,
			},
		},
	})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/sync-paths/%d/rules", syncPath.ID), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), `"enabled":false`) {
		t.Fatalf("关闭总开关响应异常: code=%d body=%s", w.Code, w.Body.String())
	}

	var enabledCount int64
	if err := db.Db.Model(&models.DirectoryUploadRule{}).
		Where("sync_path_id = ? AND enabled = ?", syncPath.ID, true).
		Count(&enabledCount).Error; err != nil {
		t.Fatalf("统计规则 enabled 状态失败: %v", err)
	}
	if enabledCount != 2 {
		t.Fatalf("总开关关闭后规则 enabled 数量 = %d，期望 2", enabledCount)
	}
	var reloaded models.SyncPath
	if err := db.Db.First(&reloaded, syncPath.ID).Error; err != nil {
		t.Fatalf("读取同步目录失败: %v", err)
	}
	if reloaded.DirectoryUploadEnabled {
		t.Fatal("同步目录目录监控总开关应关闭")
	}
	var otherEnabledCount int64
	if err := db.Db.Model(&models.DirectoryUploadRule{}).
		Where("sync_path_id = ? AND enabled = ?", otherSyncPath.ID, true).
		Count(&otherEnabledCount).Error; err != nil {
		t.Fatalf("统计其他同步目录规则失败: %v", err)
	}
	if otherEnabledCount != 1 {
		t.Fatalf("其他同步目录启用规则数量 = %d，期望 1", otherEnabledCount)
	}
}

func TestSaveDirectoryUploadSyncPathRulesAllowsSwappingExistingMonitorPaths(t *testing.T) {
	router, syncPath := setupDirectoryUploadControllerTest(t)
	monitorOne := t.TempDir()
	monitorTwo := t.TempDir()
	rules := []*models.DirectoryUploadRule{
		{
			SyncPathId:     syncPath.ID,
			AccountId:      syncPath.AccountId,
			Enabled:        true,
			MonitorPath:    monitorOne,
			RemoteRootPath: "/remote/one",
			RemoteRootId:   "remote-one",
			Recursive:      true,
		},
		{
			SyncPathId:     syncPath.ID,
			AccountId:      syncPath.AccountId,
			Enabled:        true,
			MonitorPath:    monitorTwo,
			RemoteRootPath: "/remote/two",
			RemoteRootId:   "remote-two",
			Recursive:      true,
		},
	}
	if err := db.Db.Create(&rules).Error; err != nil {
		t.Fatalf("创建目录上传规则失败: %v", err)
	}

	body, _ := json.Marshal(map[string]any{
		"enabled": true,
		"rules": []map[string]any{
			{
				"id":               rules[0].ID,
				"sync_path_id":     syncPath.ID,
				"account_id":       syncPath.AccountId,
				"enabled":          true,
				"monitor_path":     monitorTwo,
				"remote_root_path": "/remote/one",
				"remote_root_id":   "remote-one",
				"recursive":        true,
			},
			{
				"id":               rules[1].ID,
				"sync_path_id":     syncPath.ID,
				"account_id":       syncPath.AccountId,
				"enabled":          true,
				"monitor_path":     monitorOne,
				"remote_root_path": "/remote/two",
				"remote_root_id":   "remote-two",
				"recursive":        true,
			},
		},
	})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/sync-paths/%d/rules", syncPath.ID), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), `"code":200`) {
		t.Fatalf("批量保存交换规则响应异常: code=%d body=%s", w.Code, w.Body.String())
	}

	var reloaded []models.DirectoryUploadRule
	if err := db.Db.Order("id ASC").Find(&reloaded).Error; err != nil {
		t.Fatalf("读取保存后的目录上传规则失败: %v", err)
	}
	if len(reloaded) != 2 ||
		reloaded[0].MonitorPath != monitorTwo ||
		reloaded[1].MonitorPath != monitorOne {
		t.Fatalf("保存后的监控目录 = %+v，期望两条规则交换 monitor_path", reloaded)
	}
}

func TestSaveDirectoryUploadSyncPathRulesReplacesFinalSetAndStoresMasterEnabled(t *testing.T) {
	router, syncPath := setupDirectoryUploadControllerTest(t)
	syncPath.DirectoryUploadEnabled = true
	if err := db.Db.Save(syncPath).Error; err != nil {
		t.Fatalf("保存同步目录目录监控总开关失败: %v", err)
	}
	kept := &models.DirectoryUploadRule{
		SyncPathId:     syncPath.ID,
		AccountId:      syncPath.AccountId,
		Enabled:        true,
		MonitorPath:    filepath.Join(t.TempDir(), "keep"),
		RemoteRootPath: "/remote/keep",
		RemoteRootId:   "remote-keep",
		Recursive:      true,
	}
	removed := &models.DirectoryUploadRule{
		SyncPathId:     syncPath.ID,
		AccountId:      syncPath.AccountId,
		Enabled:        true,
		MonitorPath:    filepath.Join(t.TempDir(), "remove"),
		RemoteRootPath: "/remote/remove",
		RemoteRootId:   "remote-remove",
		Recursive:      true,
	}
	rules := []*models.DirectoryUploadRule{kept, removed}
	if err := db.Db.Create(&rules).Error; err != nil {
		t.Fatalf("创建目录上传规则失败: %v", err)
	}

	updatedMonitorPath := filepath.Join(t.TempDir(), "updated")
	body, _ := json.Marshal(map[string]any{
		"enabled": false,
		"rules": []map[string]any{
			{
				"id":               kept.ID,
				"sync_path_id":     syncPath.ID,
				"account_id":       syncPath.AccountId,
				"enabled":          true,
				"monitor_path":     updatedMonitorPath,
				"remote_root_path": "/remote/keep",
				"remote_root_id":   "remote-keep",
				"recursive":        true,
			},
			{
				"sync_path_id":     syncPath.ID,
				"account_id":       syncPath.AccountId,
				"enabled":          false,
				"monitor_path":     filepath.Join(t.TempDir(), "created"),
				"remote_root_path": "/remote/created",
				"remote_root_id":   "remote-created",
				"recursive":        true,
			},
		},
	})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/sync-paths/%d/rules", syncPath.ID), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), `"code":200`) {
		t.Fatalf("最终集合保存响应异常: code=%d body=%s", w.Code, w.Body.String())
	}

	var reloaded models.SyncPath
	if err := db.Db.First(&reloaded, syncPath.ID).Error; err != nil {
		t.Fatalf("读取同步目录失败: %v", err)
	}
	if reloaded.DirectoryUploadEnabled {
		t.Fatal("目录监控总开关应被关闭")
	}
	var keptReloaded models.DirectoryUploadRule
	if err := db.Db.First(&keptReloaded, kept.ID).Error; err != nil {
		t.Fatalf("读取保留规则失败: %v", err)
	}
	if !keptReloaded.Enabled {
		t.Fatal("关闭总开关不应修改保留规则自身 enabled")
	}
	if keptReloaded.MonitorPath != updatedMonitorPath {
		t.Fatalf("保留规则监控目录 = %s，期望 %s", keptReloaded.MonitorPath, updatedMonitorPath)
	}
	var removedTotal int64
	if err := db.Db.Model(&models.DirectoryUploadRule{}).Where("id = ?", removed.ID).Count(&removedTotal).Error; err != nil {
		t.Fatalf("统计移除规则失败: %v", err)
	}
	if removedTotal != 0 {
		t.Fatalf("移出最终集合的规则数量 = %d，期望 0", removedTotal)
	}
}

func TestSaveDirectoryUploadSyncPathRulesRejectsRuleFromOtherSyncPath(t *testing.T) {
	router, syncPath := setupDirectoryUploadControllerTest(t)
	otherSyncPath := &models.SyncPath{
		SourceType: models.SourceType115,
		AccountId:  syncPath.AccountId,
		BaseCid:    "other-root",
		LocalPath:  filepath.Join(t.TempDir(), "other-strm"),
		RemotePath: "/other",
	}
	if err := db.Db.Create(otherSyncPath).Error; err != nil {
		t.Fatalf("创建其他同步目录失败: %v", err)
	}
	otherRule := &models.DirectoryUploadRule{
		SyncPathId:     otherSyncPath.ID,
		AccountId:      syncPath.AccountId,
		Enabled:        true,
		MonitorPath:    t.TempDir(),
		RemoteRootPath: "/other/uploads",
		RemoteRootId:   "other-root",
		Recursive:      true,
	}
	if err := db.Db.Create(otherRule).Error; err != nil {
		t.Fatalf("创建其他同步目录规则失败: %v", err)
	}

	body, _ := json.Marshal(map[string]any{
		"enabled": true,
		"rules": []map[string]any{
			{
				"id":               otherRule.ID,
				"sync_path_id":     syncPath.ID,
				"account_id":       syncPath.AccountId,
				"enabled":          true,
				"monitor_path":     t.TempDir(),
				"remote_root_path": "/remote/uploads",
				"remote_root_id":   "remote-root",
				"recursive":        true,
			},
		},
	})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/sync-paths/%d/rules", syncPath.ID), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest || !strings.Contains(w.Body.String(), "不属于当前同步目录") {
		t.Fatalf("跨同步目录规则响应异常: code=%d body=%s", w.Code, w.Body.String())
	}
}

func TestDirectoryUploadRuntimeStatusReturnsItems(t *testing.T) {
	router, syncPath := setupDirectoryUploadControllerTest(t)
	directoryupload.StopDirectoryUploadService()
	t.Cleanup(directoryupload.StopDirectoryUploadService)

	monitorPath := t.TempDir()
	rule := &models.DirectoryUploadRule{
		SyncPathId:                    syncPath.ID,
		AccountId:                     syncPath.AccountId,
		Enabled:                       true,
		MonitorPath:                   monitorPath,
		RemoteRootPath:                "/remote",
		RemoteRootId:                  "remote-root",
		Recursive:                     true,
		WatchMode:                     models.DirectoryUploadWatchModePolling,
		StartupScanEnabled:            false,
		StabilitySeconds:              0,
		StabilityCheckIntervalSeconds: 1,
		StabilityRequiredCount:        1,
		ProcessedCacheTTLSeconds:      600,
	}
	if err := db.Db.Create(rule).Error; err != nil {
		t.Fatalf("创建目录上传规则失败: %v", err)
	}
	directoryupload.InitDirectoryUploadService()

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/runtime-status", nil)
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("运行状态响应状态异常: code=%d body=%s", w.Code, w.Body.String())
	}

	var response struct {
		Code int `json:"code"`
		Data struct {
			Items []struct {
				RuleID         uint   `json:"rule_id"`
				ConfiguredMode string `json:"configured_mode"`
				ActualMode     string `json:"actual_mode"`
				PendingCount   int    `json:"pending_count"`
			} `json:"items"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("解析运行状态响应失败: %v, body=%s", err, w.Body.String())
	}
	if response.Code != int(Success) || len(response.Data.Items) != 1 {
		t.Fatalf("运行状态响应 = %+v，期望返回 1 条成功数据", response)
	}
	item := response.Data.Items[0]
	if item.RuleID != rule.ID ||
		item.ConfiguredMode != string(models.DirectoryUploadWatchModePolling) ||
		item.ActualMode != "polling" ||
		item.PendingCount != 0 {
		t.Fatalf("运行状态 item=%+v，期望 polling 规则状态", item)
	}
}

func TestScanDirectoryUploadSyncPathRulesRejectsDisabledRules(t *testing.T) {
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
	rule.Enabled = false
	if err := db.Db.Save(rule).Error; err != nil {
		t.Fatalf("停用目录上传规则失败: %v", err)
	}

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/sync-paths/%d/scan", syncPath.ID), nil)
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK ||
		!strings.Contains(w.Body.String(), `"code":500`) ||
		!strings.Contains(w.Body.String(), "未启用") {
		t.Fatalf("停用规则扫描响应异常: code=%d body=%s", w.Code, w.Body.String())
	}
}

func TestSaveDirectoryUploadSyncPathRulesRejectsOverlappingEnabledRule(t *testing.T) {
	router, syncPath := setupDirectoryUploadControllerTest(t)
	parent := t.TempDir()
	child := filepath.Join(parent, "child")
	if err := os.MkdirAll(child, 0o755); err != nil {
		t.Fatalf("创建子监控目录失败: %v", err)
	}
	rules := []*models.DirectoryUploadRule{
		{
			SyncPathId:     syncPath.ID,
			AccountId:      syncPath.AccountId,
			Enabled:        true,
			MonitorPath:    parent,
			RemoteRootPath: "/remote/parent",
			RemoteRootId:   "remote-parent",
			Recursive:      true,
		},
		{
			SyncPathId:     syncPath.ID,
			AccountId:      syncPath.AccountId,
			Enabled:        false,
			MonitorPath:    child,
			RemoteRootPath: "/remote/child",
			RemoteRootId:   "remote-child",
			Recursive:      true,
		},
	}
	if err := db.Db.Create(&rules).Error; err != nil {
		t.Fatalf("创建目录上传规则失败: %v", err)
	}
	if err := db.Db.Model(rules[1]).Update("enabled", false).Error; err != nil {
		t.Fatalf("停用子目录规则失败: %v", err)
	}
	body, _ := json.Marshal(map[string]any{
		"enabled": true,
		"rules": []map[string]any{
			{
				"id":               rules[0].ID,
				"sync_path_id":     syncPath.ID,
				"account_id":       syncPath.AccountId,
				"enabled":          true,
				"monitor_path":     parent,
				"remote_root_path": "/remote/parent",
				"remote_root_id":   "remote-parent",
				"recursive":        true,
			},
			{
				"id":               rules[1].ID,
				"sync_path_id":     syncPath.ID,
				"account_id":       syncPath.AccountId,
				"enabled":          true,
				"monitor_path":     child,
				"remote_root_path": "/remote/child",
				"remote_root_id":   "remote-child",
				"recursive":        true,
			},
		},
	})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(
		http.MethodPut,
		fmt.Sprintf("/sync-paths/%d/rules", syncPath.ID),
		bytes.NewReader(body),
	)
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK ||
		!strings.Contains(w.Body.String(), `"code":500`) ||
		!strings.Contains(w.Body.String(), "重叠") {
		t.Fatalf("启用重叠规则响应异常: code=%d body=%s", w.Code, w.Body.String())
	}

	var childRule models.DirectoryUploadRule
	if err := db.Db.First(&childRule, rules[1].ID).Error; err != nil {
		t.Fatalf("读取子目录规则失败: %v", err)
	}
	if childRule.Enabled {
		t.Fatal("重叠规则不应被启用")
	}
}

func TestSaveDirectoryUploadSyncPathRulesRejectsPathOutsideSyncRemote(t *testing.T) {
	router, syncPath := setupDirectoryUploadControllerTest(t)
	body, _ := json.Marshal(map[string]any{
		"enabled": true,
		"rules": []map[string]any{
			{
				"sync_path_id":     syncPath.ID,
				"account_id":       syncPath.AccountId,
				"monitor_path":     t.TempDir(),
				"remote_root_path": "/other",
				"remote_root_id":   "remote-root",
			},
		},
	})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/sync-paths/%d/rules", syncPath.ID), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest || !strings.Contains(w.Body.String(), "不在同步远端目录") {
		t.Fatalf("非法远端目录响应异常: code=%d body=%s", w.Code, w.Body.String())
	}
}

func TestSaveDirectoryUploadSyncPathRulesRejectsDuplicateScope(t *testing.T) {
	router, syncPath := setupDirectoryUploadControllerTest(t)
	monitorPath := t.TempDir()
	body, _ := json.Marshal(map[string]any{
		"enabled": true,
		"rules": []map[string]any{
			{
				"sync_path_id":     syncPath.ID,
				"account_id":       syncPath.AccountId,
				"monitor_path":     monitorPath,
				"remote_root_path": "/remote/uploads",
				"remote_root_id":   "remote-root",
			},
			{
				"sync_path_id":     syncPath.ID,
				"account_id":       syncPath.AccountId,
				"monitor_path":     monitorPath,
				"remote_root_path": "/remote/uploads",
				"remote_root_id":   "remote-root",
			},
		},
	})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/sync-paths/%d/rules", syncPath.ID), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), "重复") {
		t.Fatalf("重复规则响应异常: code=%d body=%s", w.Code, w.Body.String())
	}
}

func TestSaveDirectoryUploadSyncPathRulesRejectsRecursiveOverlappingMonitorPath(t *testing.T) {
	router, syncPath := setupDirectoryUploadControllerTest(t)
	parent := t.TempDir()
	child := filepath.Join(parent, "child")
	if err := os.MkdirAll(child, 0o755); err != nil {
		t.Fatalf("创建子监控目录失败: %v", err)
	}
	body, _ := json.Marshal(map[string]any{
		"enabled": true,
		"rules": []map[string]any{
			{
				"sync_path_id":     syncPath.ID,
				"account_id":       syncPath.AccountId,
				"monitor_path":     parent,
				"remote_root_path": "/remote/parent",
				"remote_root_id":   "remote-parent",
				"recursive":        true,
			},
			{
				"sync_path_id":     syncPath.ID,
				"account_id":       syncPath.AccountId,
				"monitor_path":     child,
				"remote_root_path": "/remote/child",
				"remote_root_id":   "remote-child",
				"recursive":        true,
			},
		},
	})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/sync-paths/%d/rules", syncPath.ID), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), "重叠") {
		t.Fatalf("重叠监控目录响应异常: code=%d body=%s", w.Code, w.Body.String())
	}
}

func TestDirectoryUploadControllerRejectsInvalidID(t *testing.T) {
	router, _ := setupDirectoryUploadControllerTest(t)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/sync-paths/abc/scan", nil)
	router.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("非法 ID 状态码 = %d，期望 %d", w.Code, http.StatusBadRequest)
	}
}

func TestDirectoryUploadRuleDefaults(t *testing.T) {
	router, syncPath := setupDirectoryUploadControllerTest(t)
	body, _ := json.Marshal(map[string]any{
		"enabled": true,
		"rules": []map[string]any{
			{
				"sync_path_id":     syncPath.ID,
				"account_id":       syncPath.AccountId,
				"monitor_path":     t.TempDir(),
				"remote_root_path": "/remote",
				"remote_root_id":   "remote-root",
			},
		},
	})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/sync-paths/%d/rules", syncPath.ID), bytes.NewReader(body))
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
	body, _ := json.Marshal(map[string]any{
		"enabled": true,
		"rules": []map[string]any{
			{
				"sync_path_id":                     syncPath.ID,
				"account_id":                       syncPath.AccountId,
				"monitor_path":                     t.TempDir(),
				"remote_root_path":                 "/remote",
				"remote_root_id":                   "remote-root",
				"stability_seconds":                3600,
				"stability_check_interval_seconds": 99,
				"stability_required_count":         99,
				"rescan_interval_seconds":          999,
			},
		},
	})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/sync-paths/%d/rules", syncPath.ID), bytes.NewReader(body))
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
	body, _ := json.Marshal(map[string]any{
		"enabled": true,
		"rules": []map[string]any{
			{
				"sync_path_id":     syncPath.ID,
				"account_id":       syncPath.AccountId,
				"monitor_path":     t.TempDir(),
				"remote_root_path": "/remote",
				"remote_root_id":   "remote-root",
				"watch_mode":       "fsnotify",
				"overwrite_mode":   "replace_conflict",
			},
		},
	})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/sync-paths/%d/rules", syncPath.ID), bytes.NewReader(body))
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
			rule := map[string]any{
				"sync_path_id":     syncPath.ID,
				"account_id":       syncPath.AccountId,
				"monitor_path":     t.TempDir(),
				"remote_root_path": "/remote",
				"remote_root_id":   "remote-root",
			}
			tt.mutate(rule)
			body, _ := json.Marshal(map[string]any{
				"enabled": true,
				"rules":   []map[string]any{rule},
			})
			w := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/sync-paths/%d/rules", syncPath.ID), bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			router.ServeHTTP(w, req)
			if w.Code != http.StatusBadRequest || !strings.Contains(w.Body.String(), "不支持") {
				t.Fatalf("旧枚举响应异常: code=%d body=%s", w.Code, w.Body.String())
			}
		})
	}
}

func TestScanDirectoryUploadSyncPathRulesWithMissingSyncPath(t *testing.T) {
	router, _ := setupDirectoryUploadControllerTest(t)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/sync-paths/999/scan", nil)
	router.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("缺失同步目录状态码 = %d，期望 %d", w.Code, http.StatusNotFound)
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
	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/sync-paths/%d/scan", syncPath.ID), nil)
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), `"accepted":1`) {
		t.Fatalf("扫描响应异常: code=%d body=%s", w.Code, w.Body.String())
	}
}
