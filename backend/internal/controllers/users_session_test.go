package controllers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"

	"qmediasync/internal/db"
	"qmediasync/internal/helpers"
	"qmediasync/internal/models"
)

func setupSessionActionTest(t *testing.T) *models.User {
	t.Helper()
	gin.SetMode(gin.TestMode)
	helpers.GlobalConfig.JwtSecret = "test-secret"
	testDb, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("打开测试数据库失败: %v", err)
	}
	db.Db = testDb
	if err := db.Db.AutoMigrate(&models.User{}, &models.UserSession{}); err != nil {
		t.Fatalf("迁移用户表失败: %v", err)
	}
	user := &models.User{Username: "admin", Password: "hashed"}
	if err := db.Db.Create(user).Error; err != nil {
		t.Fatalf("创建测试用户失败: %v", err)
	}
	return user
}

func setupLoginActionTestWithPassword(t *testing.T, username string, password string) *models.User {
	t.Helper()

	user := setupSessionActionTest(t)
	hash, err := models.HashUserPassword(password)
	if err != nil {
		t.Fatalf("生成密码哈希失败: %v", err)
	}
	user.Username = username
	user.Password = hash
	if err := db.Db.Save(user).Error; err != nil {
		t.Fatalf("保存用户失败: %v", err)
	}
	return user
}

func TestSessionAction_FromCookie(t *testing.T) {
	user := setupSessionActionTest(t)
	session, csrfToken, err := models.CreateUserSession(models.CreateUserSessionInput{
		UserID:    user.ID,
		Username:  user.Username,
		ExpiresAt: time.Now().Add(time.Hour).Unix(),
	})
	if err != nil {
		t.Fatalf("创建 session 失败: %v", err)
	}
	tokenString := buildSessionCookieTokenForTest(t, session)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest(http.MethodGet, "/api/session", nil)
	req.AddCookie(&http.Cookie{Name: authCookieName, Value: tokenString})
	req.AddCookie(&http.Cookie{Name: csrfCookieName, Value: csrfToken})
	c.Request = req

	SessionAction(c)

	if w.Code != http.StatusOK {
		t.Fatalf("HTTP 状态码 = %d，期望 %d", w.Code, http.StatusOK)
	}
	body := w.Body.String()
	if strings.Contains(body, `"token"`) {
		t.Fatalf("会话响应不应返回 JWT token: %s", body)
	}
	if !strings.Contains(body, `"csrf_token"`) || !strings.Contains(body, `"username":"admin"`) {
		t.Fatalf("响应未返回用户和 csrf: %s", body)
	}
}

func TestLoginActionDoesNotReturnJWTToken(t *testing.T) {
	setupLoginActionTestWithPassword(t, "admin", "admin123")
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	body := strings.NewReader(`{"username":"admin","password":"admin123","rememberMe":false}`)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/login", body)
	c.Request.Header.Set("Content-Type", "application/json")

	LoginAction(c)

	if w.Code != http.StatusOK {
		t.Fatalf("HTTP = %d, body=%s", w.Code, w.Body.String())
	}
	if strings.Contains(w.Body.String(), `"token"`) {
		t.Fatalf("登录响应不应返回 JWT token: %s", w.Body.String())
	}
	if !strings.Contains(w.Body.String(), `"csrf_token"`) {
		t.Fatalf("登录响应应返回 csrf_token: %s", w.Body.String())
	}
}

func TestLogoutAction_ClearsCookie(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/logout", nil)

	LogoutAction(c)

	cookies := w.Result().Cookies()
	if len(cookies) == 0 || cookies[0].Name != authCookieName || cookies[0].Value != "" || cookies[0].MaxAge >= 0 {
		t.Fatalf("退出登录未清理 auth_token Cookie: %#v", cookies)
	}
}
