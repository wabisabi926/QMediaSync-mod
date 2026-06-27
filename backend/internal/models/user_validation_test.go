package models

import "testing"

func TestValidateUserUsername(t *testing.T) {
	tests := []struct {
		name     string
		username string
		wantErr  bool
	}{
		{name: "英文用户名允许", username: "admin"},
		{name: "英文数字用户名允许", username: "admin123"},
		{name: "中文用户名不允许", username: "管理员", wantErr: true},
		{name: "下划线用户名不允许", username: "admin_user", wantErr: true},
		{name: "连字符用户名不允许", username: "admin-user", wantErr: true},
		{name: "用户名过短不允许", username: "ab", wantErr: true},
		{name: "用户名过长不允许", username: "abcdefghijklmnopqrstu", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateUserUsername(tt.username)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ValidateUserUsername() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateUserPassword(t *testing.T) {
	tests := []struct {
		name    string
		pass    string
		wantErr bool
	}{
		{name: "空密码不允许", pass: "", wantErr: true},
		{name: "五位密码不允许", pass: "12345", wantErr: true},
		{name: "纯数字密码不允许", pass: "123456", wantErr: true},
		{name: "纯字母密码不允许", pass: "secret", wantErr: true},
		{name: "字母和数字组合允许", pass: "secret1", wantErr: false},
		{name: "数字和符号组合允许", pass: "12345!", wantErr: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateUserPassword(tt.pass)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ValidateUserPassword() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestUserPasswordBcryptCostIsStrong(t *testing.T) {
	if UserPasswordBcryptCost < 12 {
		t.Fatalf("UserPasswordBcryptCost = %d, want >= 12", UserPasswordBcryptCost)
	}
}
