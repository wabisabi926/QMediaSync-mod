package controllers

import (
	"Q115-STRM/internal/db"
	"Q115-STRM/internal/models"
	"bytes"
	"encoding/json"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func setupAccountControllerTest(t *testing.T) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	testDb, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("打开测试数据库失败: %v", err)
	}
	db.Db = testDb
	if err := db.Db.AutoMigrate(&models.Account{}); err != nil {
		t.Fatalf("迁移账号表失败: %v", err)
	}
}

func TestGetAccountList_ReturnsCustom115AppNameAndAppId(t *testing.T) {
	setupAccountControllerTest(t)
	account := models.Account{
		Name:       "家庭账号",
		SourceType: models.SourceType115,
		AppId:      "custom-app-id",
		AppIdName:  "家庭影音",
	}
	if err := db.Db.Create(&account).Error; err != nil {
		t.Fatalf("创建测试账号失败: %v", err)
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/account/list", nil)

	GetAccountList(c)

	if w.Code != http.StatusOK {
		t.Fatalf("HTTP 状态码 = %d，期望 %d", w.Code, http.StatusOK)
	}
	body := w.Body.String()
	if !strings.Contains(body, `"app_id_name":"家庭影音"`) {
		t.Fatalf("自定义应用名未返回: %s", body)
	}
	if !strings.Contains(body, `"app_id":"custom-app-id"`) {
		t.Fatalf("自定义 APPID 未返回: %s", body)
	}
}

func TestUpdateAccountInfo_UpdatesRemarkAndCustomAppNameOnly(t *testing.T) {
	setupAccountControllerTest(t)
	account := models.Account{
		Name:         "旧备注",
		SourceType:   models.SourceType115,
		AppId:        "custom-app-id",
		AppIdName:    "旧应用",
		Token:        "token",
		RefreshToken: "refresh-token",
		UserId:       "115-user",
		Username:     "115-name",
	}
	if err := db.Db.Create(&account).Error; err != nil {
		t.Fatalf("创建测试账号失败: %v", err)
	}

	payload, err := json.Marshal(map[string]any{
		"id":          account.ID,
		"name":        "新备注",
		"app_id_name": "新应用",
	})
	if err != nil {
		t.Fatalf("构造请求失败: %v", err)
	}
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/account/update", bytes.NewReader(payload))
	c.Request.Header.Set("Content-Type", "application/json")

	UpdateAccountInfo(c)

	if w.Code != http.StatusOK {
		t.Fatalf("HTTP 状态码 = %d，期望 %d", w.Code, http.StatusOK)
	}
	if !strings.Contains(w.Body.String(), `"code":200`) {
		t.Fatalf("响应未返回成功 code: %s", w.Body.String())
	}

	var updated models.Account
	if err := db.Db.First(&updated, account.ID).Error; err != nil {
		t.Fatalf("查询更新后账号失败: %v", err)
	}
	if updated.Name != "新备注" {
		t.Fatalf("账号备注 = %q，期望 新备注", updated.Name)
	}
	if updated.AppIdName != "新应用" {
		t.Fatalf("应用名 = %q，期望 新应用", updated.AppIdName)
	}
	if updated.AppId != "custom-app-id" || updated.Token != "token" ||
		updated.RefreshToken != "refresh-token" || updated.UserId != "115-user" ||
		updated.Username != "115-name" {
		t.Fatalf("更新账号资料影响了授权字段: %#v", updated)
	}
}

func TestCreateTmpAccountV115AuthSource(t *testing.T) {
	setupAccountControllerTest(t)
	cases := []struct {
		name          string
		payload       map[string]any
		wantAppID     string
		wantAppIDName string
	}{
		{
			name: "内置 APPID 保存数字 APPID",
			payload: map[string]any{
				"source_type":      "115",
				"name":             "qms",
				"auth_source_type": "built_in_appid",
				"auth_provider":    "official_pkce",
				"app_id":           "100197849",
				"app_id_name":      "QMediaSync",
			},
			wantAppID:     "100197849",
			wantAppIDName: "QMediaSync",
		},
		{
			name: "内置中转保存兼容字符串",
			payload: map[string]any{
				"source_type":      "115",
				"name":             "relay",
				"auth_source_type": "built_in_relay",
				"auth_provider":    "qmediasync",
				"app_id_name":      "QMediaSync",
			},
			wantAppID:     "QMediaSync",
			wantAppIDName: "QMediaSync",
		},
		{
			name: "自定义 APPID 保存用户输入",
			payload: map[string]any{
				"source_type":      "115",
				"name":             "custom",
				"auth_source_type": "custom_appid",
				"auth_provider":    "official_pkce",
				"app_id":           "100000000",
				"custom_app_name":  "我的应用",
			},
			wantAppID:     "100000000",
			wantAppIDName: "我的应用",
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			body, err := json.Marshal(tt.payload)
			if err != nil {
				t.Fatalf("构造请求失败: %v", err)
			}
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest(http.MethodPost, "/account/add", bytes.NewReader(body))
			c.Request.Header.Set("Content-Type", "application/json")

			CreateTmpAccount(c)

			if w.Code != http.StatusOK {
				t.Fatalf("HTTP 状态码 = %d，期望 %d", w.Code, http.StatusOK)
			}
			if !strings.Contains(w.Body.String(), `"code":200`) {
				t.Fatalf("响应未返回成功 code: %s", w.Body.String())
			}
			var account models.Account
			if err := db.Db.Where("name = ?", tt.payload["name"]).First(&account).Error; err != nil {
				t.Fatalf("查询创建后账号失败: %v", err)
			}
			if account.AppId != tt.wantAppID {
				t.Fatalf("app_id = %q，期望 %q", account.AppId, tt.wantAppID)
			}
			if account.AppIdName != tt.wantAppIDName {
				t.Fatalf("app_id_name = %q，期望 %q", account.AppIdName, tt.wantAppIDName)
			}
		})
	}

	body, err := json.Marshal(map[string]any{
		"source_type":      "115",
		"name":             "bad",
		"auth_source_type": "unknown",
		"auth_provider":    "official_pkce",
	})
	if err != nil {
		t.Fatalf("构造请求失败: %v", err)
	}
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/account/add", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	CreateTmpAccount(c)

	if w.Code != http.StatusOK {
		t.Fatalf("HTTP 状态码 = %d，期望 %d", w.Code, http.StatusOK)
	}
	if !strings.Contains(w.Body.String(), `"code":500`) {
		t.Fatalf("不支持的授权来源未返回 BadRequest: %s", w.Body.String())
	}
}
