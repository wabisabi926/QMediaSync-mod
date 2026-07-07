package controllers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"qmediasync/internal/db"
	"qmediasync/internal/helpers"
	"qmediasync/internal/models"

	"github.com/gin-gonic/gin"
)

func setupUserSessionsRouter(t *testing.T) (*gin.Engine, *models.User, *models.UserSession, string, string) {
	t.Helper()

	gin.SetMode(gin.TestMode)
	helpers.GlobalConfig.JwtSecret = "test-secret"
	setupControllerTestDB(t, &models.User{}, &models.UserSession{})
	user := &models.User{Username: "admin", Password: "hashed"}
	if err := db.Db.Create(user).Error; err != nil {
		t.Fatalf("创建用户失败: %v", err)
	}
	current, csrf, err := models.CreateUserSession(models.CreateUserSessionInput{UserID: user.ID, Username: user.Username, ExpiresAt: time.Now().Add(time.Hour).Unix()})
	if err != nil {
		t.Fatalf("创建当前 session 失败: %v", err)
	}
	other, _, err := models.CreateUserSession(models.CreateUserSessionInput{UserID: user.ID, Username: user.Username, UserAgent: "other", IPAddress: "127.0.0.2", ExpiresAt: time.Now().Add(time.Hour).Unix()})
	if err != nil {
		t.Fatalf("创建其他 session 失败: %v", err)
	}

	r := gin.New()
	r.Use(JWTAuthMiddleware())
	r.GET("/sessions", ListUserSessions)
	r.DELETE("/sessions/:session_id", RevokeUserSessionAction)
	r.POST("/sessions/revoke-others", RevokeOtherUserSessionsAction)
	return r, user, current, other.SessionID, csrf
}

func TestListUserSessionsMarksCurrentSession(t *testing.T) {
	r, _, current, _, csrf := setupUserSessionsRouter(t)
	tokenString := buildSessionCookieTokenForTest(t, current)
	req := httptest.NewRequest(http.MethodGet, "/sessions", nil)
	req.AddCookie(&http.Cookie{Name: authCookieName, Value: tokenString})
	req.AddCookie(&http.Cookie{Name: csrfCookieName, Value: csrf})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("HTTP = %d, body=%s", w.Code, w.Body.String())
	}
	body := w.Body.String()
	if !strings.Contains(body, `"current":true`) || !strings.Contains(body, `"current":false`) {
		t.Fatalf("响应应标记当前和其他 session: %s", body)
	}
}

func TestRevokeOtherSession(t *testing.T) {
	r, user, current, otherSessionID, csrf := setupUserSessionsRouter(t)
	tokenString := buildSessionCookieTokenForTest(t, current)
	req := httptest.NewRequest(http.MethodDelete, "/sessions/"+otherSessionID, nil)
	req.Host = "localhost:12333"
	req.Header.Set("Origin", "http://localhost:12333")
	req.Header.Set(csrfHeaderName, csrf)
	req.AddCookie(&http.Cookie{Name: authCookieName, Value: tokenString})
	req.AddCookie(&http.Cookie{Name: csrfCookieName, Value: csrf})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("HTTP = %d, body=%s", w.Code, w.Body.String())
	}
	sessions, err := models.ListActiveUserSessions(user.ID, time.Now().Unix())
	if err != nil {
		t.Fatalf("查询 session 失败: %v", err)
	}
	if len(sessions) != 1 || sessions[0].SessionID != current.SessionID {
		t.Fatalf("应只剩当前 session: %#v", sessions)
	}
}
