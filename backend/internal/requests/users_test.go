package requests

import "testing"

func TestUserSettingsRequestValidate(t *testing.T) {
	tests := []struct {
		name    string
		req     ChangeUserCredentialRequest
		wantErr bool
	}{
		{name: "只修改用户名通过", req: ChangeUserCredentialRequest{Username: "admin"}},
		{name: "修改用户名和密码通过", req: ChangeUserCredentialRequest{Username: "admin", NewPassword: "secret1"}},
		{name: "用户名为空失败", req: ChangeUserCredentialRequest{Username: " "}, wantErr: true},
		{name: "用户名过短失败", req: ChangeUserCredentialRequest{Username: "ab"}, wantErr: true},
		{name: "用户名过长失败", req: ChangeUserCredentialRequest{Username: "abcdefghijklmnopqrstu"}, wantErr: true},
		{name: "密码过短失败", req: ChangeUserCredentialRequest{Username: "admin", NewPassword: "12345"}, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.req.Validate()
			if (err != nil) != tt.wantErr {
				t.Fatalf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
