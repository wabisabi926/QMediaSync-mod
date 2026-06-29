package controllers

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"qmediasync/internal/db"
	"qmediasync/internal/helpers"
	"qmediasync/internal/models"
	"qmediasync/internal/v115open"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func setupQueueStatsControllerTest(t *testing.T) *gin.Engine {
	t.Helper()

	gin.SetMode(gin.TestMode)
	helpers.AppLogger = &helpers.QLogger{Logger: log.New(io.Discard, "", 0)}
	helpers.V115Log = &helpers.QLogger{Logger: log.New(io.Discard, "", 0)}
	t.Cleanup(func() {
		v115open.GetGlobalExecutor().Stop()
	})

	testDB, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("打开测试数据库失败: %v", err)
	}
	db.Db = testDB

	if err := db.Db.AutoMigrate(&models.RequestStat{}); err != nil {
		t.Fatalf("创建请求统计表失败: %v", err)
	}

	r := gin.New()
	r.GET("/115/queue/stats", GetQueueStats)
	return r
}

func TestGetQueueStats重启后从数据库返回请求统计(t *testing.T) {
	r := setupQueueStatsControllerTest(t)
	now := time.Now().Unix()
	stats := []models.RequestStat{
		{RequestTime: now - 1, Duration: 100, IsThrottled: false},
		{RequestTime: now - 30, Duration: 200, IsThrottled: true},
	}
	if err := db.Db.Create(&stats).Error; err != nil {
		t.Fatalf("创建请求统计失败: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/115/queue/stats?time_window=3600", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("HTTP status = %d，期望 200，body=%s", w.Code, w.Body.String())
	}

	var resp APIResponse[map[string]any]
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}
	if resp.Code != Success {
		t.Fatalf("Code = %d，期望 %d，message=%s", resp.Code, Success, resp.Message)
	}
	if got := int64(resp.Data["total_requests"].(float64)); got != 2 {
		t.Fatalf("total_requests = %d，期望 2", got)
	}
	if got := int64(resp.Data["qpm_count"].(float64)); got != 2 {
		t.Fatalf("qpm_count = %d，期望 2", got)
	}
	if got := int64(resp.Data["throttled_count"].(float64)); got != 1 {
		t.Fatalf("throttled_count = %d，期望 1", got)
	}
}
