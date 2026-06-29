package models

import (
	"testing"

	"qmediasync/internal/db"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func setupRequestStatTestDB(t *testing.T) {
	t.Helper()

	testDB, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("打开测试数据库失败: %v", err)
	}
	db.Db = testDB

	if err := db.Db.AutoMigrate(&RequestStat{}); err != nil {
		t.Fatalf("创建请求统计表失败: %v", err)
	}
}

func TestGetRequestStatsWindow按时间窗口聚合请求统计(t *testing.T) {
	setupRequestStatTestDB(t)

	now := int64(1_800_000_000)
	stats := []RequestStat{
		{RequestTime: now - 1, Duration: 100, IsThrottled: false},
		{RequestTime: now - 30, Duration: 200, IsThrottled: true},
		{RequestTime: now - 120, Duration: 300, IsThrottled: false},
		{RequestTime: now - 3700, Duration: 400, IsThrottled: true},
	}
	if err := db.Db.Create(&stats).Error; err != nil {
		t.Fatalf("创建请求统计失败: %v", err)
	}

	got, err := GetRequestStatsWindow(now, 3600)
	if err != nil {
		t.Fatalf("GetRequestStatsWindow() error = %v", err)
	}

	if got.TotalRequests != 3 {
		t.Fatalf("TotalRequests = %d，期望 3", got.TotalRequests)
	}
	if got.QPSCount != 1 {
		t.Fatalf("QPSCount = %d，期望 1", got.QPSCount)
	}
	if got.QPMCount != 2 {
		t.Fatalf("QPMCount = %d，期望 2", got.QPMCount)
	}
	if got.QPHCount != 3 {
		t.Fatalf("QPHCount = %d，期望 3", got.QPHCount)
	}
	if got.ThrottledCount != 1 {
		t.Fatalf("ThrottledCount = %d，期望 1", got.ThrottledCount)
	}
	if got.AvgResponseTimeMS != 200 {
		t.Fatalf("AvgResponseTimeMS = %d，期望 200", got.AvgResponseTimeMS)
	}
}
