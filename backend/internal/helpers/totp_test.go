package helpers

import (
	"testing"
	"time"
)

func TestTOTPValidateCode(t *testing.T) {
	secret, otpURL, err := GenerateTOTPSecret("QMediaSync", "admin")
	if err != nil {
		t.Fatalf("生成 TOTP Secret 失败: %v", err)
	}
	if secret == "" {
		t.Fatal("secret 不应为空")
	}
	if otpURL == "" {
		t.Fatal("otpauth URL 不应为空")
	}

	code, err := GenerateTOTPCodeForTest(secret, time.Now())
	if err != nil {
		t.Fatalf("生成测试验证码失败: %v", err)
	}
	if !ValidateTOTPCode(secret, code) {
		t.Fatal("正确验证码应校验通过")
	}
	if ValidateTOTPCode(secret, "") {
		t.Fatal("空验证码不应校验通过")
	}
	if ValidateTOTPCode(secret, "000000") {
		t.Fatal("错误验证码不应校验通过")
	}
}
