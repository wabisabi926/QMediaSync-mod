package controllers

import (
	"bytes"
	"encoding/json"
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

func setupLogSettingControllerTest(t *testing.T) (*gin.Engine, *bytes.Buffer) {
	t.Helper()

	gin.SetMode(gin.TestMode)

	oldConfigDir := helpers.ConfigDir
	oldGlobalConfig := helpers.GlobalConfig
	oldLogLevel := helpers.ConfiguredLogLevel()
	oldAppLogger := helpers.AppLogger
	oldV115Log := helpers.V115Log
	oldOpenListLog := helpers.OpenListLog
	oldBaiduPanLog := helpers.BaiduPanLog
	oldTMDBLog := helpers.TMDBLog
	t.Cleanup(func() {
		helpers.ConfigDir = oldConfigDir
		helpers.GlobalConfig = oldGlobalConfig
		helpers.SetGlobalLogLevel(oldLogLevel)
		helpers.AppLogger = oldAppLogger
		helpers.V115Log = oldV115Log
		helpers.OpenListLog = oldOpenListLog
		helpers.BaiduPanLog = oldBaiduPanLog
		helpers.TMDBLog = oldTMDBLog
	})

	helpers.ConfigDir = t.TempDir()
	helpers.GlobalConfig = *helpers.MakeDefaultConfig()
	helpers.SetGlobalLogLevel(helpers.LogLevelInfo)
	if err := helpers.SaveConfig(&helpers.GlobalConfig); err != nil {
		t.Fatalf("保存测试配置失败: %v", err)
	}

	var logBuf bytes.Buffer
	helpers.AppLogger = &helpers.QLogger{Logger: log.New(&logBuf, "", 0)}
	helpers.V115Log = &helpers.QLogger{Logger: log.New(&bytes.Buffer{}, "", 0)}
	helpers.OpenListLog = &helpers.QLogger{Logger: log.New(&bytes.Buffer{}, "", 0)}
	helpers.BaiduPanLog = &helpers.QLogger{Logger: log.New(&bytes.Buffer{}, "", 0)}
	helpers.TMDBLog = &helpers.QLogger{Logger: log.New(&bytes.Buffer{}, "", 0)}

	r := gin.New()
	r.GET("/setting/log", GetLogSetting)
	r.POST("/setting/log", UpdateLogSetting)
	return r, &logBuf
}

func TestGetLogSetting返回当前日志等级(t *testing.T) {
	r, _ := setupLogSettingControllerTest(t)
	helpers.SetGlobalLogLevel(helpers.LogLevelWarn)

	req := httptest.NewRequest(http.MethodGet, "/setting/log", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("HTTP = %d, body=%s", w.Code, w.Body.String())
	}
	var resp APIResponse[LogSettingResponse]
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}
	if resp.Code != Success {
		t.Fatalf("Code = %d, want %d", resp.Code, Success)
	}
	if resp.Data.Level != "warn" {
		t.Fatalf("level = %q, want warn", resp.Data.Level)
	}
	if strings.Join(resp.Data.Levels, ",") != "debug,info,warn,error" {
		t.Fatalf("levels = %v", resp.Data.Levels)
	}
}

func TestUpdateLogSetting保存后立即生效(t *testing.T) {
	r, logBuf := setupLogSettingControllerTest(t)

	req := httptest.NewRequest(http.MethodPost, "/setting/log", strings.NewReader(`{"level":"debug"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("HTTP = %d, body=%s", w.Code, w.Body.String())
	}
	var resp APIResponse[LogSettingResponse]
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}
	if resp.Code != Success {
		t.Fatalf("Code = %d, message=%s", resp.Code, resp.Message)
	}
	if helpers.GlobalConfig.Log.Level != "debug" {
		t.Fatalf("GlobalConfig.Log.Level = %q, want debug", helpers.GlobalConfig.Log.Level)
	}

	saved, err := os.ReadFile(filepath.Join(helpers.ConfigDir, "config.yaml"))
	if err != nil {
		t.Fatalf("读取保存后的配置失败: %v", err)
	}
	if !strings.Contains(string(saved), "level: debug") {
		t.Fatalf("配置文件未保存日志等级: %s", string(saved))
	}

	helpers.AppLogger.Debug("debug message")
	if got := logBuf.String(); !strings.Contains(got, "[DEBUG] debug message") {
		t.Fatalf("保存后运行中的 logger 未立即切换到 debug: %s", got)
	}
}

func TestUpdateLogSetting拒绝非法日志等级(t *testing.T) {
	cases := []struct {
		name string
		body string
	}{
		{name: "未知等级", body: `{"level":"verbose"}`},
		{name: "空白等级", body: `{"level":" "}`},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r, _ := setupLogSettingControllerTest(t)

			req := httptest.NewRequest(http.MethodPost, "/setting/log", strings.NewReader(tc.body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			r.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Fatalf("HTTP = %d, body=%s", w.Code, w.Body.String())
			}
			var resp APIResponse[any]
			if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
				t.Fatalf("解析响应失败: %v", err)
			}
			if resp.Code != BadRequest {
				t.Fatalf("Code = %d, want %d", resp.Code, BadRequest)
			}
			if helpers.GlobalConfig.Log.Level != "info" {
				t.Fatalf("非法保存不应修改日志等级，got %q", helpers.GlobalConfig.Log.Level)
			}
		})
	}
}
