package validation

import (
	"fmt"
	"strings"
	"unicode"
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
	if !isASCIIAlnum(username) {
		return New(field, "只能包含英文和数字")
	}
	return nil
}

// UserPassword 校验系统用户的密码规则。
func UserPassword(field string, password string) error {
	if utf8.RuneCountInString(password) < MinUserPasswordLength {
		return New(field, fmt.Sprintf("长度不能小于 %d", MinUserPasswordLength))
	}
	if isPureDigits(password) || isPureLetters(password) {
		return New(field, "不能是纯数字或纯字母")
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

// LoginCredentials 校验登录输入是否完整，不套用新凭据创建规则。
func LoginCredentials(usernameField string, username string, passwordField string, password string) error {
	username = strings.TrimSpace(username)
	if username == "" {
		return New(usernameField, "不能为空")
	}
	if utf8.RuneCountInString(username) > MaxUserUsernameLength {
		return New(usernameField, fmt.Sprintf("长度不能超过 %d", MaxUserUsernameLength))
	}
	if password == "" {
		return New(passwordField, "不能为空")
	}
	return nil
}

func isASCIIAlnum(value string) bool {
	for _, item := range value {
		if item >= 'a' && item <= 'z' {
			continue
		}
		if item >= 'A' && item <= 'Z' {
			continue
		}
		if item >= '0' && item <= '9' {
			continue
		}
		return false
	}
	return value != ""
}

func isPureDigits(value string) bool {
	for _, item := range value {
		if !unicode.IsDigit(item) {
			return false
		}
	}
	return value != ""
}

func isPureLetters(value string) bool {
	for _, item := range value {
		if !unicode.IsLetter(item) {
			return false
		}
	}
	return value != ""
}
