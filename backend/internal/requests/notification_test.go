package requests

import "testing"

func TestNotificationRequestValidate(t *testing.T) {
	t.Run("Telegram 必填字段通过", func(t *testing.T) {
		req := CreateTelegramChannelRequest{ChannelName: "telegram", BotToken: "token", ChatID: "chat"}
		if err := req.Validate(); err != nil {
			t.Fatalf("Validate() error = %v", err)
		}
	})

	t.Run("Telegram 空白渠道名失败", func(t *testing.T) {
		req := CreateTelegramChannelRequest{ChannelName: " ", BotToken: "token", ChatID: "chat"}
		if err := req.Validate(); err == nil {
			t.Fatal("Validate() error = nil, want error")
		}
	})

	t.Run("MeoW 非法 endpoint 失败", func(t *testing.T) {
		req := CreateMeoWChannelRequest{ChannelName: "meow", Nickname: "qms", Endpoint: "ftp://example.com"}
		if err := req.Validate(); err == nil {
			t.Fatal("Validate() error = nil, want error")
		}
	})

	t.Run("Bark 非法 icon 失败", func(t *testing.T) {
		req := CreateBarkChannelRequest{ChannelName: "bark", DeviceKey: "key", Icon: "bad"}
		if err := req.Validate(); err == nil {
			t.Fatal("Validate() error = nil, want error")
		}
	})

	t.Run("ServerChan 默认 endpoint 通过", func(t *testing.T) {
		req := CreateServerChanChannelRequest{ChannelName: "serverchan", SCKEY: "key"}
		if err := req.Validate(); err != nil {
			t.Fatalf("Validate() error = %v", err)
		}
	})

	t.Run("Webhook 创建通过并规范化", func(t *testing.T) {
		req := CustomWebhookChannelRequest{
			ChannelName: "webhook",
			Endpoint:    "https://example.com/hook",
			Method:      "post",
			Format:      "JSON",
			Template:    `{"text":"{{title}}"}`,
			AuthType:    "Bearer",
			AuthToken:   "token",
		}
		if err := req.ValidateCreate(); err != nil {
			t.Fatalf("ValidateCreate() error = %v", err)
		}
		if req.Method != "POST" || req.Format != "json" || req.AuthType != "bearer" || req.QueryParam != "q" {
			t.Fatalf("ValidateCreate() normalized request = %+v", req)
		}
	})

	t.Run("Webhook 创建非法方法失败", func(t *testing.T) {
		req := CustomWebhookChannelRequest{
			ChannelName: "webhook",
			Endpoint:    "https://example.com/hook",
			Method:      "PUT",
			Template:    "hello",
		}
		if err := req.ValidateCreate(); err == nil {
			t.Fatal("ValidateCreate() error = nil, want error")
		}
	})

	t.Run("Webhook Bearer 缺少 token 失败", func(t *testing.T) {
		req := CustomWebhookChannelRequest{
			ChannelName: "webhook",
			Endpoint:    "https://example.com/hook",
			Method:      "POST",
			Format:      "text",
			Template:    "hello",
			AuthType:    "bearer",
		}
		if err := req.ValidateCreate(); err == nil {
			t.Fatal("ValidateCreate() error = nil, want error")
		}
	})

	t.Run("Webhook JSON 模板错误失败", func(t *testing.T) {
		req := CustomWebhookChannelRequest{
			ChannelName: "webhook",
			Endpoint:    "https://example.com/hook",
			Method:      "POST",
			Format:      "json",
			Template:    "{",
		}
		if err := req.ValidateCreate(); err == nil {
			t.Fatal("ValidateCreate() error = nil, want error")
		}
	})

	t.Run("Webhook 多个额外 Header 通过并规范化头名", func(t *testing.T) {
		req := CustomWebhookChannelRequest{
			ChannelName: "webhook",
			Endpoint:    "https://example.com/hook",
			Method:      "POST",
			Format:      "text",
			Template:    "hello",
			Headers: map[string]string{
				" X-Trace-ID ":     "trace-1",
				"X-Webhook-Source": "qmediasync",
			},
		}
		if err := req.ValidateCreate(); err != nil {
			t.Fatalf("ValidateCreate() error = %v", err)
		}
		if _, ok := req.Headers[" X-Trace-ID "]; ok {
			t.Fatal("ValidateCreate() 未规范化 Header 名称")
		}
		if req.Headers["X-Trace-ID"] != "trace-1" || req.Headers["X-Webhook-Source"] != "qmediasync" {
			t.Fatalf("Headers = %#v", req.Headers)
		}
	})

	t.Run("Webhook 空 Header 名失败", func(t *testing.T) {
		req := CustomWebhookChannelRequest{
			ChannelName: "webhook",
			Endpoint:    "https://example.com/hook",
			Method:      "POST",
			Format:      "text",
			Template:    "hello",
			Headers:     map[string]string{" ": "bad"},
		}
		if err := req.ValidateCreate(); err == nil {
			t.Fatal("ValidateCreate() error = nil, want error")
		}
	})

	t.Run("Webhook 更新缺少 ID 失败", func(t *testing.T) {
		req := CustomWebhookChannelRequest{Endpoint: "https://example.com/hook"}
		if err := req.ValidateUpdate("GET", ""); err == nil {
			t.Fatal("ValidateUpdate() error = nil, want error")
		}
	})
}
