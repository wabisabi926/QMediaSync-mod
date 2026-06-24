package models

import "fmt"

const MinUserPasswordLength = 6

// ValidateUserPassword 校验管理员密码长度。
func ValidateUserPassword(password string) error {
	if len(password) < MinUserPasswordLength {
		return fmt.Errorf("密码长度至少%d个字符", MinUserPasswordLength)
	}
	return nil
}
