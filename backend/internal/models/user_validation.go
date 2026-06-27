package models

import "qmediasync/internal/validation"

const MinUserUsernameLength = validation.MinUserUsernameLength
const MaxUserUsernameLength = validation.MaxUserUsernameLength
const MinUserPasswordLength = validation.MinUserPasswordLength
const UserPasswordBcryptCost = 12

// ValidateUserUsername 校验管理员用户名长度。
func ValidateUserUsername(username string) error {
	return validation.UserUsername("username", username)
}

// ValidateUserPassword 校验管理员密码长度。
func ValidateUserPassword(password string) error {
	return validation.UserPassword("password", password)
}

// ValidateUserCredentials 校验管理员用户名和密码。
func ValidateUserCredentials(username string, password string) error {
	return validation.UserCredentials("username", username, "password", password)
}
