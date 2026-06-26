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

type testBackgroundHandler struct {
	channelType string
	stopCount   int
}

func (h *testBackgroundHandler) Send(context.Context, *notification.Notification) error {
	return nil
}

func (h *testBackgroundHandler) GetChannelType() string {
	if h.channelType == "" {
		return "test"
	}
	return h.channelType
}

func (h *testBackgroundHandler) IsHealthy() bool {
	return true
}

func (h *testBackgroundHandler) Start(context.Context) {}

func (h *testBackgroundHandler) Stop() {
	h.stopCount++
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

func TestReloadRulesPreservesBackgroundHandlers(t *testing.T) {
	manager, testDb, _ := setupNotificationManagerTest(t)
	channel := notification.NotificationChannel{ChannelType: "telegram", ChannelName: "Telegram", IsEnabled: true}
	if err := testDb.Create(&channel).Error; err != nil {
		t.Fatalf("创建通知渠道失败: %v", err)
	}
	handler := &testBackgroundHandler{channelType: "telegram"}
	manager.handlers[channel.ID] = &channelInfo{handler: handler, config: &channel}
	rule := notification.NotificationRule{
		ChannelID: channel.ID,
		EventType: string(notification.SyncFinished),
		IsEnabled: true,
	}
	if err := testDb.Create(&rule).Error; err != nil {
		t.Fatalf("创建通知规则失败: %v", err)
	}

	manager.ReloadRules()

	manager.mu.RLock()
	gotHandler := manager.handlers[channel.ID].handler
	gotRuleIDs := append([]uint(nil), manager.rules[string(notification.SyncFinished)]...)
	manager.mu.RUnlock()
	if gotHandler != handler {
		t.Fatal("刷新通知规则不应替换已有后台 handler")
	}
	if handler.stopCount != 0 {
		t.Fatalf("刷新通知规则不应停止后台 handler，实际停止次数: %d", handler.stopCount)
	}
	if len(gotRuleIDs) != 1 || gotRuleIDs[0] != channel.ID {
		t.Fatalf("sync_finish 规则路由 = %v，期望仅包含渠道 %d", gotRuleIDs, channel.ID)
	}
}

func TestRemoveChannelStopsBackgroundHandlerAndDropsRules(t *testing.T) {
	manager, testDb, _ := setupNotificationManagerTest(t)
	channel := notification.NotificationChannel{ChannelType: "telegram", ChannelName: "Telegram", IsEnabled: true}
	if err := testDb.Create(&channel).Error; err != nil {
		t.Fatalf("创建通知渠道失败: %v", err)
	}
	handler := &testBackgroundHandler{channelType: "telegram"}
	manager.handlers[channel.ID] = &channelInfo{handler: handler, config: &channel}
	if err := testDb.Create(&notification.NotificationRule{
		ChannelID: channel.ID,
		EventType: string(notification.SyncFinished),
		IsEnabled: true,
	}).Error; err != nil {
		t.Fatalf("创建通知规则失败: %v", err)
	}
	manager.ReloadRules()

	manager.RemoveChannel(channel.ID)

	manager.mu.RLock()
	_, handlerExists := manager.handlers[channel.ID]
	gotRuleIDs := append([]uint(nil), manager.rules[string(notification.SyncFinished)]...)
	manager.mu.RUnlock()
	if handlerExists {
		t.Fatal("卸载通知渠道后不应保留 handler")
	}
	if handler.stopCount != 1 {
		t.Fatalf("卸载通知渠道应停止一次后台 handler，实际停止次数: %d", handler.stopCount)
	}
	if len(gotRuleIDs) != 0 {
		t.Fatalf("卸载通知渠道后不应保留规则路由，实际: %v", gotRuleIDs)
	}
}

func TestTelegramChannelHandlerStopCancelsListeningContext(t *testing.T) {
	handler := &TelegramChannelHandler{}
	ctx, cancel := context.WithCancel(context.Background())
	handler.listenCancel = cancel

	handler.Stop()
	handler.Stop()

	select {
	case <-ctx.Done():
	case <-time.After(time.Second):
		t.Fatal("停止 Telegram handler 应取消监听 context")
	}
}

func TestTelegramChannelHandlerStartInitializesBotInBackground(t *testing.T) {
	initStarted := make(chan struct{})
	allowInitReturn := make(chan struct{})
	listeningStarted := make(chan struct{}, 1)
	handler := &TelegramChannelHandler{
		initBotFunc: func() error {
			close(initStarted)
			<-allowInitReturn
			return nil
		},
		startListeningFunc: func(context.Context, map[string]func([]string) helpers.CommandResponse) {
			listeningStarted <- struct{}{}
		},
	}

	done := make(chan struct{})
	go func() {
		handler.Start(context.Background())
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Telegram handler Start 不应等待 Bot 初始化完成")
	}
	select {
	case <-initStarted:
	case <-time.After(time.Second):
		t.Fatal("Telegram handler Start 应在后台启动 Bot 初始化")
	}

	close(allowInitReturn)
	select {
	case <-listeningStarted:
	case <-time.After(time.Second):
		t.Fatal("Bot 初始化完成后应进入监听")
	}
}

func TestTelegramChannelHandlerStopBeforeInitCompletesSkipsListening(t *testing.T) {
	initStarted := make(chan struct{})
	allowInitReturn := make(chan struct{})
	initReturned := make(chan struct{})
	listeningStarted := make(chan struct{}, 1)
	handler := &TelegramChannelHandler{
		initBotFunc: func() error {
			close(initStarted)
			<-allowInitReturn
			close(initReturned)
			return nil
		},
		startListeningFunc: func(context.Context, map[string]func([]string) helpers.CommandResponse) {
			listeningStarted <- struct{}{}
		},
	}

	handler.Start(context.Background())
	select {
	case <-initStarted:
	case <-time.After(time.Second):
		t.Fatal("Telegram handler Start 应在后台启动 Bot 初始化")
	}

	handler.Stop()
	close(allowInitReturn)
	select {
	case <-initReturned:
	case <-time.After(time.Second):
		t.Fatal("测试中的 Bot 初始化未按预期结束")
	}
	select {
	case <-listeningStarted:
		t.Fatal("停止 Telegram handler 后，不应在初始化完成后继续进入监听")
	case <-time.After(100 * time.Millisecond):
	}
}
