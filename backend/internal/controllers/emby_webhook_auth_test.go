package controllers

import (
	"bytes"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"

	"qmediasync/internal/db"
	"qmediasync/internal/helpers"
	"qmediasync/internal/models"

	"github.com/gin-gonic/gin"
)

func setupWebhookAPIKeyTest(t *testing.T) (*gin.Engine, string) {
	t.Helper()

	gin.SetMode(gin.TestMode)
	helpers.AppLogger = &helpers.QLogger{Logger: log.New(io.Discard, "", 0)}
	setupControllerTestDB(t, &models.User{}, &models.ApiKey{})
	user := &models.User{Username: "admin", Password: "hashed"}
	if err := db.Db.Create(user).Error; err != nil {
		t.Fatalf("创建用户失败: %v", err)
	}
	_, rawKey, err := models.CreateAPIKey(user.ID, "emby webhook")
	if err != nil {
		t.Fatalf("创建 API Key 失败: %v", err)
	}
	models.GlobalEmbyConfig = &models.EmbyConfig{
		EmbyUrl:    "http://emby.example",
		EmbyApiKey: "emby-api-key",
		EnableAuth: 1,
	}

	r := gin.New()
	r.POST("/emby/webhook", Webhook)
	return r, rawKey
}

func TestWebhookAPIKey支持Header并保留Query(t *testing.T) {
	cases := []struct {
		name       string
		setAPIKey  func(req *http.Request, rawKey string)
		requestURI string
	}{
		{
			name:       "通过 X-API-Key 请求头鉴权",
			requestURI: "/emby/webhook",
			setAPIKey: func(req *http.Request, rawKey string) {
				req.Header.Set(apiKeyHeaderName, rawKey)
			},
		},
		{
			name:       "兼容 api_key 查询参数鉴权",
			requestURI: "/emby/webhook",
			setAPIKey: func(req *http.Request, rawKey string) {
				req.URL.RawQuery = "api_key=" + rawKey
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r, rawKey := setupWebhookAPIKeyTest(t)
			req := httptest.NewRequest(http.MethodPost, tc.requestURI, bytes.NewBufferString(`{"Event":"system.test"}`))
			tc.setAPIKey(req, rawKey)
			w := httptest.NewRecorder()

			r.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Fatalf("HTTP = %d, body=%s", w.Code, w.Body.String())
			}
		})
	}
}
