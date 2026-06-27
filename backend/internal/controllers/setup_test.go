package controllers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"qmediasync/internal/db"
	"qmediasync/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func setupInitialSetupControllerTest(t *testing.T) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	testDb, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("打开测试数据库失败: %v", err)
	}
	db.Db = testDb
	if err := db.Db.AutoMigrate(&models.User{}); err != nil {
		t.Fatalf("迁移用户表失败: %v", err)
	}
	configureInitialSetupTokenForTest("setup-token")
	t.Cleanup(func() {
		DisableInitialSetup()
	})
}

func TestSetupStatusActionRequired(t *testing.T) {
	setupInitialSetupControllerTest(t)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/setup/status", nil)

	SetupStatusAction(c)

	if w.Code != http.StatusOK {
		t.Fatalf("HTTP = %d，期望 200", w.Code)
	}
	if !strings.Contains(w.Body.String(), `"required":true`) {
		t.Fatalf("初始化状态响应错误: %s", w.Body.String())
	}
	if strings.Contains(w.Body.String(), "setup-token") {
		t.Fatalf("初始化状态不应泄露 token: %s", w.Body.String())
	}
}

func TestCreateInitialAdminActionRejectsBadToken(t *testing.T) {
	setupInitialSetupControllerTest(t)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	body := strings.NewReader(`{"setup_token":"bad-token","username":"admin","password":"admin123"}`)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/setup/admin", body)
	c.Request.Header.Set("Content-Type", "application/json")

	CreateInitialAdminAction(c)

	if w.Code != http.StatusOK {
		t.Fatalf("HTTP = %d，期望 200", w.Code)
	}
	if !strings.Contains(w.Body.String(), "初始化码无效") {
		t.Fatalf("错误信息不正确: %s", w.Body.String())
	}
}

func TestCreateInitialAdminActionCreatesUserAndDisablesSetup(t *testing.T) {
	setupInitialSetupControllerTest(t)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	body := strings.NewReader(`{"setup_token":"setup-token","username":"admin","password":"admin123"}`)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/setup/admin", body)
	c.Request.Header.Set("Content-Type", "application/json")

	CreateInitialAdminAction(c)

	if w.Code != http.StatusOK {
		t.Fatalf("HTTP = %d，期望 200", w.Code)
	}
	if !strings.Contains(w.Body.String(), `"code":200`) {
		t.Fatalf("创建首个管理员响应错误: %s", w.Body.String())
	}

	var count int64
	if err := db.Db.Model(&models.User{}).Count(&count).Error; err != nil {
		t.Fatalf("统计用户失败: %v", err)
	}
	if count != 1 {
		t.Fatalf("用户数量 = %d，期望 1", count)
	}

	w = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/setup/status", nil)
	SetupStatusAction(c)
	if !strings.Contains(w.Body.String(), `"required":false`) {
		t.Fatalf("创建后初始化状态应关闭: %s", w.Body.String())
	}
}
