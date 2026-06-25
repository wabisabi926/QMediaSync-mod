package controllers

import (
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"Q115-STRM/internal/db"
	"Q115-STRM/internal/helpers"
	"Q115-STRM/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func setupAuthSecurityTest(t *testing.T) (*gin.Engine, *models.User, *models.UserSession, string) {
	t.Helper()

	gin.SetMode(gin.TestMode)
	helpers.AppLogger = &helpers.QLogger{Logger: log.New(io.Discard, "", 0)}
	helpers.GlobalConfig.JwtSecret = "test-secret"
	testDb, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("打开测试数据库失败: %v", err)
	}
	db.Db = testDb
	if err := db.Db.AutoMigrate(&models.User{}, &models.UserSession{}, &models.ApiKey{}); err != nil {
		t.Fatalf("迁移测试表失败: %v", err)
	}
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

func TestAPIKeyHeaderSkipsCSRFAndWinsOverQuery(t *testing.T) {
	r, user, _, _ := setupAuthSecurityTest(t)
	_, rawHeaderKey, err := models.CreateAPIKey(user.ID, "header")
	if err != nil {
		t.Fatalf("创建 header api key 失败: %v", err)
	}
	otherUser := &models.User{Username: "other", Password: "hashed"}
	if err := db.Db.Create(otherUser).Error; err != nil {
		t.Fatalf("创建其他用户失败: %v", err)
	}
	_, rawQueryKey, err := models.CreateAPIKey(otherUser.ID, "query")
	if err != nil {
		t.Fatalf("创建 query api key 失败: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/protected?api_key="+rawQueryKey, nil)
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
