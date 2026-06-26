package requests

import (
	"strings"
	"unicode/utf8"

	"qmediasync/internal/validation"
)

// ChangeUserCredentialRequest 修改用户凭据请求。
type ChangeUserCredentialRequest struct {
	Username    string `json:"username" form:"username"`
	NewPassword string `json:"new_password" form:"new_password"`
}

// Validate 校验用户凭据修改请求。
func (r ChangeUserCredentialRequest) Validate() error {
	username := strings.TrimSpace(r.Username)
	if username == "" {
		return validation.New("username", "不能为空")
	}
	usernameLen := utf8.RuneCountInString(username)
	if usernameLen < 3 || usernameLen > 20 {
		return validation.New("username", "长度超出允许范围")
	}
	if r.NewPassword != "" && utf8.RuneCountInString(r.NewPassword) < 6 {
		return validation.New("new_password", "长度不能小于 6")
	}
	return nil
}
