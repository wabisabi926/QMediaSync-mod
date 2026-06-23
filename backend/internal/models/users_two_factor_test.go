package models

import "testing"

func TestUserTwoFactorState(t *testing.T) {
	user := &User{Username: "admin"}
	if user.IsTwoFactorEnabled() {
		t.Fatal("默认不应启用 2FA")
	}

	user.EnableTwoFactor("encrypted-secret")
	if !user.IsTwoFactorEnabled() {
		t.Fatal("启用后应返回 true")
	}
	if user.TwoFactorSecret != "encrypted-secret" {
		t.Fatal("应保存加密后的 secret")
	}

	user.DisableTwoFactor()
	if user.IsTwoFactorEnabled() {
		t.Fatal("关闭后应返回 false")
	}
	if user.TwoFactorSecret != "" {
		t.Fatal("关闭后应清空 secret")
	}
}
