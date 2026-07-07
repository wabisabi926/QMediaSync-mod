package controllers

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"qmediasync/internal/db"
	"qmediasync/internal/directoryupload"
	"qmediasync/internal/helpers"
	"qmediasync/internal/models"

	"github.com/gin-gonic/gin"
)

func TestUpdateSyncPathReloadsDirectoryUploadServiceWhenMasterSwitchChanges(t *testing.T) {
	gin.SetMode(gin.TestMode)
	helpers.AppLogger = &helpers.QLogger{Logger: log.New(io.Discard, "", 0)}
	setupControllerTestDB(
		t,
		&models.Account{},
		&models.SyncPath{},
		&models.DirectoryUploadRule{},
		&models.DirectoryUploadProcessedFile{},
	)
	models.SettingsGlobal = &models.Settings{
		SettingStrm: models.SettingStrm{VideoExtArr: []string{".mkv"}, MinVideoSize: 0},
	}
	t.Cleanup(directoryupload.StopDirectoryUploadService)

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
		SettingStrm:            models.SettingStrm{VideoExtArr: []string{".mkv"}, MinVideoSize: 0},
	}
	if err := db.Db.Create(syncPath).Error; err != nil {
		t.Fatalf("创建同步目录失败: %v", err)
	}
	rule := &models.DirectoryUploadRule{
		SyncPathId:               syncPath.ID,
		AccountId:                account.ID,
		Enabled:                  true,
		MonitorPath:              t.TempDir(),
		RemoteRootPath:           "/remote/uploads",
		RemoteRootId:             "remote-root",
		Recursive:                true,
		WatchMode:                models.DirectoryUploadWatchModePolling,
		StartupScanEnabled:       false,
		ProcessedCacheTTLSeconds: 600,
	}
	if err := db.Db.Create(rule).Error; err != nil {
		t.Fatalf("创建目录监控规则失败: %v", err)
	}

	directoryupload.InitDirectoryUploadService()
	if got := len(directoryupload.GetDirectoryUploadRuntimeStatuses()); got != 1 {
		t.Fatalf("关闭前运行规则数量 = %d，期望 1", got)
	}

	router := gin.New()
	router.POST("/sync/path-update", UpdateSyncPath)
	body, _ := json.Marshal(map[string]any{
		"id":                       syncPath.ID,
		"source_type":              string(models.SourceType115),
		"account_id":               account.ID,
		"base_cid":                 syncPath.BaseCid,
		"local_path":               syncPath.LocalPath,
		"remote_path":              syncPath.RemotePath,
		"enable_cron":              false,
		"custom_config":            false,
		"directory_upload_enabled": false,
	})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/sync/path-update", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK || !bytes.Contains(w.Body.Bytes(), []byte(`"code":200`)) {
		t.Fatalf("更新同步目录响应异常: code=%d body=%s", w.Code, w.Body.String())
	}
	if got := len(directoryupload.GetDirectoryUploadRuntimeStatuses()); got != 0 {
		t.Fatalf("关闭总开关后运行规则数量 = %d，期望服务重载后为 0", got)
	}
}
