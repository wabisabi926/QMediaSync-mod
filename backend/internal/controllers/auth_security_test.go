package controllers

import (
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"qmediasync/internal/db"
	"qmediasync/internal/helpers"
	"qmediasync/internal/models"

	"github.com/gin-gonic/gin"
	"gopkg.in/yaml.v2"
)

func setupAuthSecurityTest(t *testing.T) (*gin.Engine, *models.User, *models.UserSession, string) {
	t.Helper()

	gin.SetMode(gin.TestMode)
	helpers.AppLogger = &helpers.QLogger{Logger: log.New(io.Discard, "", 0)}
	helpers.GlobalConfig = helpers.Config{JwtSecret: "test-secret"}
	setupControllerTestDB(t, &models.User{}, &models.UserSession{}, &models.ApiKey{})
	user := &models.User{Username: "admin", Password: "hashed"}
	if err := db.Db.Create(user).Error; err != nil {
		t.Fatalf("创建用户失败: %v", err)
	}
	session, csrfToken, err := models.CreateUserSession(models.CreateUserSessionInput{
		UserID:    user.ID,
		Username:  user.Username,
		UserAgent: "test-agent",
		IPAddress: "127.0.0.1",
		ExpiresAt: time.Now().Add(time.Hour).Unix(),
	})
	if err != nil {
		t.Fatalf("创建 session 失败: %v", err)
	}

	r := gin.New()
	r.Use(JWTAuthMiddleware())
	r.POST("/protected", func(c *gin.Context) {
		user, ok := CurrentUser(c)
		if !ok || user.ID == 0 {
			c.JSON(http.StatusInternalServerError, gin.H{"message": "missing user"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"username": user.Username, "auth_method": CurrentAuthMethod(c)})
	})
	return r, user, session, csrfToken
}

func TestCorsRestrictsCredentialedOrigins(t *testing.T) {
	gin.SetMode(gin.TestMode)
	helpers.GlobalConfig = helpers.Config{}
	r := gin.New()
	r.Use(Cors())
	r.GET("/protected", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Host = "localhost:12333"
	req.Header.Set("Origin", "https://evil.example")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("不可信 Origin 不应被允许，got %q", got)
	}
	if got := w.Header().Get("Access-Control-Allow-Credentials"); got != "" {
		t.Fatalf("不可信 Origin 不应允许携带凭证，got %q", got)
	}
}

func TestCorsAllowsConfiguredTrustedOrigin(t *testing.T) {
	gin.SetMode(gin.TestMode)
	helpers.GlobalConfig = helpers.Config{}
	if err := yaml.Unmarshal([]byte("trustedOrigins:\n  - https://qms.example.com\n"), &helpers.GlobalConfig); err != nil {
		t.Fatalf("解析 trustedOrigins 失败: %v", err)
	}
	r := gin.New()
	r.Use(Cors())
	r.GET("/protected", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Host = "api.example.com"
	req.Header.Set("Origin", "https://qms.example.com")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "https://qms.example.com" {
		t.Fatalf("配置的可信 Origin 应被允许，got %q", got)
	}
	if got := w.Header().Get("Access-Control-Allow-Credentials"); got != "true" {
		t.Fatalf("配置的可信 Origin 应允许携带凭证，got %q", got)
	}
}

func TestCorsAllowsConfiguredTrustedOriginWithDefaultPort(t *testing.T) {
	cases := []struct {
		name            string
		trustedOrigin   string
		browserOrigin   string
		requestHost     string
		wantAllowOrigin string
	}{
		{
			name:            "HTTPS 默认端口",
			trustedOrigin:   "https://qms.example.com:443",
			browserOrigin:   "https://qms.example.com",
			requestHost:     "api.example.com",
			wantAllowOrigin: "https://qms.example.com",
		},
		{
			name:            "HTTP 默认端口",
			trustedOrigin:   "http://localhost:80",
			browserOrigin:   "http://localhost",
			requestHost:     "api.example.com",
			wantAllowOrigin: "http://localhost",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			helpers.GlobalConfig = helpers.Config{}
			if err := yaml.Unmarshal([]byte("trustedOrigins:\n  - "+tc.trustedOrigin+"\n"), &helpers.GlobalConfig); err != nil {
				t.Fatalf("解析 trustedOrigins 失败: %v", err)
			}
			r := gin.New()
			r.Use(Cors())
			r.GET("/protected", func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"ok": true})
			})

			req := httptest.NewRequest(http.MethodGet, "/protected", nil)
			req.Host = tc.requestHost
			req.Header.Set("Origin", tc.browserOrigin)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			if got := w.Header().Get("Access-Control-Allow-Origin"); got != tc.wantAllowOrigin {
				t.Fatalf("带默认端口的可信 Origin 配置应被允许，got %q", got)
			}
		})
	}
}

func TestCookieSessionRequiresCSRFForUnsafeMethod(t *testing.T) {
	r, _, session, csrfToken := setupAuthSecurityTest(t)
	tokenString := buildSessionCookieTokenForTest(t, session)

	req := httptest.NewRequest(http.MethodPost, "/protected", nil)
	req.Host = "localhost:12333"
	req.Header.Set("Origin", "http://localhost:12333")
	req.AddCookie(&http.Cookie{Name: authCookieName, Value: tokenString})
	req.AddCookie(&http.Cookie{Name: csrfCookieName, Value: csrfToken})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusForbidden {
		t.Fatalf("缺少 X-CSRF-Token 时 HTTP = %d, want 403", w.Code)
	}

	req = httptest.NewRequest(http.MethodPost, "/protected", nil)
	req.Host = "localhost:12333"
	req.Header.Set("Origin", "http://localhost:12333")
	req.Header.Set(csrfHeaderName, csrfToken)
	req.AddCookie(&http.Cookie{Name: authCookieName, Value: tokenString})
	req.AddCookie(&http.Cookie{Name: csrfCookieName, Value: csrfToken})
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("带 CSRF 时 HTTP = %d, body=%s", w.Code, w.Body.String())
	}
}

func TestCookieSessionAllowsDefaultViteOrigin(t *testing.T) {
	r, _, session, csrfToken := setupAuthSecurityTest(t)
	tokenString := buildSessionCookieTokenForTest(t, session)

	req := httptest.NewRequest(http.MethodPost, "/protected", nil)
	req.Host = "localhost:12333"
	req.Header.Set("Origin", "http://localhost:5173")
	req.Header.Set(csrfHeaderName, csrfToken)
	req.AddCookie(&http.Cookie{Name: authCookieName, Value: tokenString})
	req.AddCookie(&http.Cookie{Name: csrfCookieName, Value: csrfToken})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("默认 Vite 开发来源应通过 CSRF，HTTP = %d, body=%s", w.Code, w.Body.String())
	}
}

func TestCookieSessionAllowsConfiguredTrustedOrigin(t *testing.T) {
	r, _, session, csrfToken := setupAuthSecurityTest(t)
	if err := yaml.Unmarshal([]byte("trustedOrigins:\n  - https://qms.example.com\n"), &helpers.GlobalConfig); err != nil {
		t.Fatalf("解析 trustedOrigins 失败: %v", err)
	}
	tokenString := buildSessionCookieTokenForTest(t, session)

	req := httptest.NewRequest(http.MethodPost, "/protected", nil)
	req.Host = "api.example.com"
	req.Header.Set("Origin", "https://qms.example.com")
	req.Header.Set(csrfHeaderName, csrfToken)
	req.AddCookie(&http.Cookie{Name: authCookieName, Value: tokenString})
	req.AddCookie(&http.Cookie{Name: csrfCookieName, Value: csrfToken})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("配置的可信来源应通过 CSRF，HTTP = %d, body=%s", w.Code, w.Body.String())
	}
}

func TestAPIKeyHeaderSkipsCSRFAndWinsOverQuery(t *testing.T) {
	r, user, _, _ := setupAuthSecurityTest(t)
	_, rawHeaderKey, err := models.CreateAPIKey(user.ID, "header")
	if err != nil {
		t.Fatalf("创建 header api key 失败: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/protected?api_key=invalid-query-key", nil)
	req.Header.Set(apiKeyHeaderName, rawHeaderKey)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("HTTP = %d, body=%s", w.Code, w.Body.String())
	}
	if got := w.Body.String(); !strings.Contains(got, `"username":"admin"`) {
		t.Fatalf("header api key 未优先: %s", got)
	}
}

func TestAPIKeyAuthCanRunBackgroundUpdateSynchronouslyInTests(t *testing.T) {
	r, user, _, _ := setupAuthSecurityTest(t)
	setAuthBackgroundTaskRunnerForTest(t, func(fn func()) {
		fn()
	})
	apiKey, rawKey, err := models.CreateAPIKey(user.ID, "sync-update")
	if err != nil {
		t.Fatalf("创建 api key 失败: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/protected", nil)
	req.Header.Set(apiKeyHeaderName, rawKey)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("HTTP = %d, body=%s", w.Code, w.Body.String())
	}
	var updated models.ApiKey
	if err := db.Db.First(&updated, apiKey.ID).Error; err != nil {
		t.Fatalf("读取 api key 失败: %v", err)
	}
	if updated.LastUsedAt == 0 {
		t.Fatalf("同步后台执行器未更新 api key last_used_at")
	}
}

func TestCookieAuthCanRunSessionTouchSynchronouslyInTests(t *testing.T) {
	r, _, session, csrfToken := setupAuthSecurityTest(t)
	setAuthBackgroundTaskRunnerForTest(t, func(fn func()) {
		fn()
	})
	oldLastSeenAt := time.Now().Add(-2 * time.Minute).Unix()
	if err := db.Db.Model(session).Update("last_seen_at", oldLastSeenAt).Error; err != nil {
		t.Fatalf("设置旧 session last_seen_at 失败: %v", err)
	}
	tokenString := buildSessionCookieTokenForTest(t, session)

	req := httptest.NewRequest(http.MethodPost, "/protected", nil)
	req.Host = "localhost:12333"
	req.Header.Set("Origin", "http://localhost:12333")
	req.Header.Set(csrfHeaderName, csrfToken)
	req.AddCookie(&http.Cookie{Name: authCookieName, Value: tokenString})
	req.AddCookie(&http.Cookie{Name: csrfCookieName, Value: csrfToken})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("HTTP = %d, body=%s", w.Code, w.Body.String())
	}
	var updated models.UserSession
	if err := db.Db.First(&updated, session.ID).Error; err != nil {
		t.Fatalf("读取 session 失败: %v", err)
	}
	if updated.LastSeenAt == oldLastSeenAt {
		t.Fatalf("同步后台执行器未更新 session last_seen_at")
	}
}

func buildSessionCookieTokenForTest(t *testing.T, session *models.UserSession) string {
	t.Helper()

	tokenString, err := SignSessionJWT(SessionClaimsInput{
		UserID:    session.UserID,
		Username:  session.Username,
		SessionID: session.SessionID,
		TokenID:   session.TokenID,
		ExpiresAt: session.ExpiresAt,
	})
	if err != nil {
		t.Fatalf("签发测试 JWT 失败: %v", err)
	}
	return tokenString
}

func setAuthBackgroundTaskRunnerForTest(t *testing.T, runner func(func())) {
	t.Helper()
	oldRunner := runAuthBackgroundTask
	runAuthBackgroundTask = runner
	t.Cleanup(func() {
		runAuthBackgroundTask = oldRunner
	})
}
