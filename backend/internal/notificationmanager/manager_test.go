package notificationmanager

import (
	"bytes"
	"context"
	"log"
	"strings"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"

	"qmediasync/internal/helpers"
	"qmediasync/internal/notification"
)

func setupNotificationManagerTest(t *testing.T) (*EnhancedNotificationManager, *gorm.DB, *bytes.Buffer) {
	t.Helper()
	testDb, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("打开测试数据库失败: %v", err)
	}
	if err := testDb.AutoMigrate(&notification.NotificationChannel{}, &notification.NotificationRule{}, &notification.CustomWebhookChannelConfig{}); err != nil {
		t.Fatalf("迁移通知测试表失败: %v", err)
	}
	var buf bytes.Buffer
	helpers.AppLogger = &helpers.QLogger{Logger: log.New(&buf, "", 0)}
	manager := NewEnhancedNotificationManager(testDb, func() string { return "" })
	return manager, testDb, &buf
}

func TestSendNotificationWithoutChannelsIsQuiet(t *testing.T) {
	manager, _, buf := setupNotificationManagerTest(t)
	if err := manager.LoadChannels(); err != nil {
		t.Fatalf("加载渠道失败: %v", err)
	}
	notif := &notification.Notification{
		Type:      notification.SyncFinished,
		Title:     "同步完成",
		Timestamp: time.Now(),
	}
	if err := manager.SendNotification(context.Background(), notif); err != nil {
		t.Fatalf("未配置通知时应静默返回，实际错误: %v", err)
	}
	if strings.Contains(buf.String(), "未找到事件类型") {
		t.Fatalf("未配置通知时不应写未找到规则日志，实际日志：%s", buf.String())
	}
}

func TestLoadChannelsRoutesEnabledRulesForDuplicateChannelTypes(t *testing.T) {
	manager, testDb, _ := setupNotificationManagerTest(t)
	channels := []notification.NotificationChannel{
		{ChannelType: "webhook", ChannelName: "Webhook A", IsEnabled: true},
		{ChannelType: "webhook", ChannelName: "Webhook B", IsEnabled: true},
		{ChannelType: "webhook", ChannelName: "Webhook C", IsEnabled: false},
	}
	if err := testDb.Create(&channels).Error; err != nil {
		t.Fatalf("创建通知渠道失败: %v", err)
	}
	if err := testDb.Model(&notification.NotificationChannel{}).Where("id = ?", channels[2].ID).Update("is_enabled", false).Error; err != nil {
		t.Fatalf("禁用通知渠道失败: %v", err)
	}
	for _, channel := range channels {
		if err := testDb.Create(&notification.CustomWebhookChannelConfig{
			ChannelID: channel.ID,
			Endpoint:  "https://example.invalid/webhook",
			Method:    "POST",
			Template:  `{"title":"{{title}}"}`,
			Format:    "json",
		}).Error; err != nil {
			t.Fatalf("创建 webhook 配置失败: %v", err)
		}
		if err := testDb.Create(&notification.NotificationRule{
			ChannelID: channel.ID,
			EventType: string(notification.SyncFinished),
			IsEnabled: true,
		}).Error; err != nil {
			t.Fatalf("创建通知规则失败: %v", err)
		}
	}

	if err := manager.LoadChannels(); err != nil {
		t.Fatalf("加载通知渠道失败: %v", err)
	}
	manager.mu.RLock()
	defer manager.mu.RUnlock()
	if got := len(manager.rules[string(notification.SyncFinished)]); got != 2 {
		t.Fatalf("sync_finish 启用渠道数量 = %d，期望 2", got)
	}
}
