package helpers

import (
	"time"

	"github.com/pquerna/otp"
	"github.com/pquerna/otp/totp"
)

// GenerateTOTPSecret 生成 TOTP 密钥和 otpauth URL
func GenerateTOTPSecret(issuer string, accountName string) (string, string, error) {
	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      issuer,
		AccountName: accountName,
		Period:      30,
		SecretSize:  20,
		Digits:      otp.DigitsSix,
		Algorithm:   otp.AlgorithmSHA1,
	})
	if err != nil {
		return "", "", err
	}
	return key.Secret(), key.URL(), nil
}

// ValidateTOTPCode 校验 TOTP 动态验证码
func ValidateTOTPCode(secret string, code string) bool {
	if secret == "" || code == "" {
		return false
	}
	return totp.Validate(code, secret)
}

// GenerateTOTPCodeForTest 仅供测试生成指定时间的 TOTP 验证码
func GenerateTOTPCodeForTest(secret string, now time.Time) (string, error) {
	return totp.GenerateCode(secret, now)
}
