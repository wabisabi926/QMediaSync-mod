package main

import (
	"strings"
	"testing"
)

func TestValidateInitialAdminCredentials(t *testing.T) {
	tests := []struct {
		name      string
		username  string
		password  string
		wantError string
	}{
		{name: "用户名不能为空", username: "", password: "123456", wantError: "管理员用户名不能为空"},
		{name: "密码不能少于六位", username: "admin", password: "12345", wantError: "密码长度至少6个字符"},
		{name: "合法账号", username: "admin", password: "123456", wantError: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateInitialAdminCredentials(tt.username, tt.password)
			if tt.wantError == "" {
				if err != nil {
					t.Fatalf("validateInitialAdminCredentials() error = %v", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tt.wantError) {
				t.Fatalf("validateInitialAdminCredentials() error = %v, want contains %q", err, tt.wantError)
			}
		})
	}
}
