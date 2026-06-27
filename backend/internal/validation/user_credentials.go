package validation

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

const MinUserUsernameLength = 3
const MaxUserUsernameLength = 20
const MinUserPasswordLength = 6

// UserUsername 校验系统用户的用户名规则。
func UserUsername(field string, username string) error {
	username = strings.TrimSpace(username)
	if username == "" {
		return New(field, "不能为空")
	}
	usernameLen := utf8.RuneCountInString(username)
	if usernameLen < MinUserUsernameLength || usernameLen > MaxUserUsernameLength {
		return New(field, fmt.Sprintf("长度必须在 %d 到 %d 个字符之间", MinUserUsernameLength, MaxUserUsernameLength))
	}
	return nil
}

// UserPassword 校验系统用户的密码规则。
func UserPassword(field string, password string) error {
	if utf8.RuneCountInString(password) < MinUserPasswordLength {
		return New(field, fmt.Sprintf("长度不能小于 %d", MinUserPasswordLength))
	}
	return nil
}

// UserCredentials 校验系统用户的用户名和密码。
func UserCredentials(usernameField string, username string, passwordField string, password string) error {
	if err := UserUsername(usernameField, username); err != nil {
		return err
	}
	return UserPassword(passwordField, password)
}
