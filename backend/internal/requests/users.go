package requests

import "qmediasync/internal/validation"

// LoginRequest 用户登录请求。
type LoginRequest struct {
	Username   string `json:"username" form:"username"`
	Password   string `json:"password" form:"password"`
	TOTPCode   string `json:"totp_code" form:"totp_code"`
	RememberMe bool   `json:"rememberMe" form:"rememberMe"`
}

// Validate 校验用户登录请求。
func (r LoginRequest) Validate() error {
	return validation.UserCredentials("username", r.Username, "password", r.Password)
}

// EnableTwoFactorRequest 启用两步验证请求。
type EnableTwoFactorRequest struct {
	TOTPCode string `json:"totp_code" form:"totp_code"`
}

// Validate 校验启用两步验证请求。
func (r EnableTwoFactorRequest) Validate() error {
	return validation.NonBlank("totp_code", r.TOTPCode)
}

// DisableTwoFactorRequest 关闭两步验证请求。
type DisableTwoFactorRequest struct {
	Password string `json:"password" form:"password"`
	TOTPCode string `json:"totp_code" form:"totp_code"`
}

// Validate 校验关闭两步验证请求。
func (r DisableTwoFactorRequest) Validate() error {
	if err := validation.NonBlank("password", r.Password); err != nil {
		return err
	}
	return validation.NonBlank("totp_code", r.TOTPCode)
}

// ChangeUserCredentialRequest 修改用户凭据请求。
type ChangeUserCredentialRequest struct {
	Username    string `json:"username" form:"username"`
	NewPassword string `json:"new_password" form:"new_password"`
}

// Validate 校验用户凭据修改请求。
func (r ChangeUserCredentialRequest) Validate() error {
	if err := validation.UserUsername("username", r.Username); err != nil {
		return err
	}
	if r.NewPassword != "" {
		return validation.UserPassword("new_password", r.NewPassword)
	}
	return nil
}
