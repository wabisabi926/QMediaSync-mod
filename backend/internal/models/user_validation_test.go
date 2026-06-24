package models

import "testing"

func TestValidateUserPassword(t *testing.T) {
	tests := []struct {
		name    string
		pass    string
		wantErr bool
	}{
		{name: "空密码不允许", pass: "", wantErr: true},
		{name: "五位密码不允许", pass: "12345", wantErr: true},
		{name: "六位密码允许", pass: "123456", wantErr: false},
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
