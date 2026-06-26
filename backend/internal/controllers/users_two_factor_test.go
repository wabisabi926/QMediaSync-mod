package controllers

import (
	"testing"

	"qmediasync/internal/requests"
)

func TestLoginFailureMessageIsGeneric(t *testing.T) {
	msg := loginFailureMessage()
	if msg != "登录失败" {
		t.Fatalf("登录失败信息应统一为登录失败，实际为 %s", msg)
	}
}

func TestDisableTwoFactorRequiresPasswordAndCode(t *testing.T) {
	req := requests.DisableTwoFactorRequest{}
	if req.Validate() == nil {
		t.Fatal("空密码和空验证码不应允许关闭 2FA")
	}
	req.Password = "admin123"
	if req.Validate() == nil {
		t.Fatal("缺少 TOTP 验证码不应允许关闭 2FA")
	}
	req.TOTPCode = "123456"
	if req.Validate() != nil {
		t.Fatal("同时提供密码和 TOTP 验证码后请求格式应有效")
	}
}
