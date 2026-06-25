package models

import "fmt"

const MinUserPasswordLength = 6
const UserPasswordBcryptCost = 12

// ValidateUserPassword 校验管理员密码长度。
func ValidateUserPassword(password string) error {
	if len(password) < MinUserPasswordLength {
		return fmt.Errorf("密码长度至少 %d 个字符", MinUserPasswordLength)
	}
	return nil
}
