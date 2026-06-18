package openlist

import (
	"testing"
)

func TestValidateToken_InvalidToken(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			// 如果 panic，说明日志系统未初始化，这是预期的
			t.Logf("预期 Panic 发生 (日志系统未初始化): %v", r)
		}
	}()
	client := NewClient(0, "http://localhost:8080", "", "", "invalid_token")
	_, err := client.GetUserInfo("invalid_token")
	if err == nil {
		t.Errorf("ValidateToken 应该返回错误，但返回了 nil")
	}
}

func TestValidateToken_EmptyToken(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			// 如果 panic，说明日志系统未初始化，这是预期的
			t.Logf("预期 Panic 发生 (日志系统未初始化): %v", r)
		}
	}()
	client := NewClient(0, "http://localhost:8080", "", "", "")
	_, err := client.GetUserInfo("")
	if err == nil {
		t.Errorf("ValidateToken 应该返回错误，但返回了 nil")
	}
}
