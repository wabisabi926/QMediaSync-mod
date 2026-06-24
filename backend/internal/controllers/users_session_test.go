package controllers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/golang-jwt/jwt/v5"
	"gorm.io/gorm"

	"Q115-STRM/internal/db"
	"Q115-STRM/internal/helpers"
	"Q115-STRM/internal/models"
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
	if err := db.Db.AutoMigrate(&models.User{}); err != nil {
		t.Fatalf("迁移用户表失败: %v", err)
	}
	user := &models.User{Username: "admin", Password: "hashed"}
	if err := db.Db.Create(user).Error; err != nil {
		t.Fatalf("创建测试用户失败: %v", err)
	}
	return user
}

func buildSessionTestToken(t *testing.T, user *models.User) string {
	t.Helper()
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, &LoginUser{
		ID:       user.ID,
		Username: user.Username,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
		},
	})
	tokenString, err := token.SignedString([]byte(helpers.GlobalConfig.JwtSecret))
	if err != nil {
		t.Fatalf("签发测试 token 失败: %v", err)
	}
	return tokenString
}

func TestSessionAction_FromCookie(t *testing.T) {
	user := setupSessionActionTest(t)
	tokenString := buildSessionTestToken(t, user)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest(http.MethodGet, "/api/session", nil)
	req.AddCookie(&http.Cookie{Name: "auth_token", Value: tokenString})
	c.Request = req

	SessionAction(c)

	if w.Code != http.StatusOK {
		t.Fatalf("HTTP 状态码 = %d，期望 %d", w.Code, http.StatusOK)
	}
	if !strings.Contains(w.Body.String(), `"code":200`) {
		t.Fatalf("响应未返回成功 code: %s", w.Body.String())
	}
	if !strings.Contains(w.Body.String(), `"username":"admin"`) {
		t.Fatalf("响应未返回当前用户: %s", w.Body.String())
	}
}

func TestLogoutAction_ClearsCookie(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/logout", nil)

	LogoutAction(c)

	cookies := w.Result().Cookies()
	if len(cookies) == 0 || cookies[0].Name != "auth_token" || cookies[0].Value != "" || cookies[0].MaxAge >= 0 {
		t.Fatalf("退出登录未清理 auth_token Cookie: %#v", cookies)
	}
}
