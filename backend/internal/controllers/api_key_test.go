package controllers

import (
	"bytes"
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
)

func setupAPIKeyControllerTest(t *testing.T) (*models.User, *bytes.Buffer) {
	t.Helper()

	gin.SetMode(gin.TestMode)
	var logBuf bytes.Buffer
	helpers.AppLogger = &helpers.QLogger{Logger: log.New(&logBuf, "", 0)}
	setupControllerTestDB(t, &models.User{}, &models.ApiKey{})
	user := &models.User{Username: "admin", Password: "hashed"}
	if err := db.Db.Create(user).Error; err != nil {
		t.Fatalf("创建用户失败: %v", err)
	}
	return user, &logBuf
}

func TestCreateAPIKeyLogsUsernameAndKeyName(t *testing.T) {
	user, logBuf := setupAPIKeyControllerTest(t)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/api-keys", strings.NewReader(`{"name":"测试"}`))
	c.Request.Header.Set("Content-Type", "application/json")
	SetCurrentUser(c, user, authMethodSession)

	CreateAPIKey(c)

	if w.Code != http.StatusOK {
		t.Fatalf("HTTP = %d, body=%s", w.Code, w.Body.String())
	}
	logText := logBuf.String()
	if !strings.Contains(logText, "用户 admin（ID：1）创建了 API Key：测试（ID：1）") {
		t.Fatalf("创建日志未包含真实用户名和 API Key 名称: %s", logText)
	}
	if strings.Contains(logText, "qms_") {
		t.Fatalf("创建日志不应包含完整 API Key: %s", logText)
	}
}

func TestDeleteAPIKeyLogsUsernameAndKeyName(t *testing.T) {
	user, logBuf := setupAPIKeyControllerTest(t)
	helpers.AppLogger = &helpers.QLogger{Logger: log.New(io.Discard, "", 0)}
	apiKey, _, err := models.CreateAPIKey(user.ID, "测试")
	if err != nil {
		t.Fatalf("创建 API Key 失败: %v", err)
	}
	logBuf.Reset()
	helpers.AppLogger = &helpers.QLogger{Logger: log.New(logBuf, "", 0)}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodDelete, "/api/api-keys/1", nil)
	c.Params = gin.Params{{Key: "id", Value: "1"}}
	SetCurrentUser(c, user, authMethodSession)

	DeleteAPIKey(c)

	if w.Code != http.StatusOK {
		t.Fatalf("HTTP = %d, body=%s", w.Code, w.Body.String())
	}
	logText := logBuf.String()
	if !strings.Contains(logText, "用户 admin（ID：1）删除了 API Key：测试（ID：1）") {
		t.Fatalf("删除日志未包含真实用户名和 API Key 名称: %s", logText)
	}
	var count int64
	if err := db.Db.Model(&models.ApiKey{}).Where("id = ?", apiKey.ID).Count(&count).Error; err != nil {
		t.Fatalf("查询 API Key 失败: %v", err)
	}
	if count != 0 {
		t.Fatalf("API Key 未删除，剩余数量 = %d", count)
	}
}
