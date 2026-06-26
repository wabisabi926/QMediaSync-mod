package controllers

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"qmediasync/internal/helpers"
	"qmediasync/internal/models"

	"github.com/gin-gonic/gin"
)

type LoginRequest struct {
	Username   string `json:"username" form:"username"`
	Password   string `json:"password" form:"password"`
	TOTPCode   string `json:"totp_code" form:"totp_code"`
	RememberMe bool   `json:"rememberMe" form:"rememberMe"`
}

type EnableTwoFactorRequest struct {
	TOTPCode string `json:"totp_code" form:"totp_code"`
}

type DisableTwoFactorRequest struct {
	Password string `json:"password" form:"password"`
	TOTPCode string `json:"totp_code" form:"totp_code"`
}

func (req DisableTwoFactorRequest) IsValid() bool {
	return req.Password != "" && req.TOTPCode != ""
}

func loginFailureMessage() string {
	return "登录失败"
}

func writeLoginFailure(c *gin.Context) {
	c.JSON(http.StatusOK, APIResponse[any]{Code: BadRequest, Message: loginFailureMessage(), Data: nil})
}

// LoginAction 用户登录
// @Summary 用户登录
// @Description 用户登录并返回 JWT Token
// @Tags 用户管理
// @Accept json
// @Produce json
// @Param username body string true "用户名"
// @Param password body string true "密码"
// @Success 200 {object} object
// @Failure 200 {object} object
// @Router /api/login [post]
func LoginAction(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBind(&req); err != nil {
		writeLoginFailure(c)
		return
	}
	username := strings.TrimSpace(req.Username)
	password := req.Password
	if username == "" || password == "" {
		writeLoginFailure(c)
		return
	}

	clientIP := c.ClientIP()
	if allowed, wait := defaultLoginRateLimiter.Allow(clientIP, username); !allowed {
		if helpers.AppLogger != nil {
			helpers.AppLogger.Warnf("登录限流：ip=%s username=%s wait=%s", clientIP, username, wait)
		}
		writeLoginRateLimited(c, wait)
		return
	}

	user, userErr := models.CheckLogin(username, password)
	if userErr != nil || user.ID == 0 {
		defaultLoginRateLimiter.RecordFailure(clientIP, username, "password_or_user_invalid")
		writeLoginFailure(c)
		return
	}

	if user.IsTwoFactorEnabled() {
		secret, err := helpers.DecryptLocalSecret(user.TwoFactorSecret)
		if err != nil || !helpers.ValidateTOTPCode(secret, req.TOTPCode) {
			defaultLoginRateLimiter.RecordFailure(clientIP, username, "totp_invalid")
			writeLoginFailure(c)
			return
		}
	}

	// 根据 rememberMe 参数设置 token 和 Cookie 有效期
	tokenExpire := time.Hour * 24
	cookieMaxAge := 0
	if req.RememberMe {
		tokenExpire = time.Hour * 24 * 30
		cookieMaxAge = int(tokenExpire.Seconds())
	}
	expiresAt := time.Now().Add(tokenExpire).Unix()
	session, csrfToken, err := models.CreateUserSession(models.CreateUserSessionInput{
		UserID:    user.ID,
		Username:  user.Username,
		UserAgent: c.Request.UserAgent(),
		IPAddress: clientIP,
		ExpiresAt: expiresAt,
	})
	if err != nil {
		if helpers.AppLogger != nil {
			helpers.AppLogger.Errorf("创建登录会话失败：%v", err)
		}
		writeLoginFailure(c)
		return
	}
	tokenString, err := SignSessionJWT(SessionClaimsInput{
		UserID:    user.ID,
		Username:  user.Username,
		SessionID: session.SessionID,
		TokenID:   session.TokenID,
		ExpiresAt: expiresAt,
	})
	if err != nil {
		if helpers.AppLogger != nil {
			helpers.AppLogger.Errorf("登录签名失败：%v", err)
		}
		writeLoginFailure(c)
		return
	}
	setSessionCookies(c, tokenString, csrfToken, cookieMaxAge)
	defaultLoginRateLimiter.Reset(clientIP, username)

	c.JSON(http.StatusOK, APIResponse[any]{
		Code:    Success,
		Message: "登录成功",
		Data: gin.H{
			"user":       buildLoginUserResponse(user),
			"session":    buildSessionResponse(session, session.SessionID),
			"csrf_token": csrfToken,
		},
	})
}

// SessionAction 返回当前登录会话
func SessionAction(c *gin.Context) {
	cookie, err := c.Request.Cookie(authCookieName)
	if err != nil || cookie.Value == "" {
		c.JSON(http.StatusUnauthorized, APIResponse[any]{Code: BadRequest, Message: "登录凭证不存在", Data: nil})
		return
	}
	loginUser, err := ValidateJWT(cookie.Value)
	if err != nil {
		c.JSON(http.StatusUnauthorized, APIResponse[any]{Code: BadRequest, Message: "登录凭证无效", Data: nil})
		return
	}
	session, err := models.GetActiveUserSession(loginUser.SessionID, time.Now().Unix())
	if err != nil || session.UserID != loginUser.ID {
		c.JSON(http.StatusUnauthorized, APIResponse[any]{Code: BadRequest, Message: "登录会话已失效", Data: nil})
		return
	}
	user, err := models.GetUserById(session.UserID)
	if err != nil || user.ID == 0 {
		c.JSON(http.StatusUnauthorized, APIResponse[any]{Code: BadRequest, Message: "登录用户不存在", Data: nil})
		return
	}
	csrfCookie, err := c.Request.Cookie(csrfCookieName)
	if err != nil || csrfCookie.Value == "" || !session.ValidateCSRFToken(csrfCookie.Value) {
		csrfToken, csrfErr := models.GenerateSessionSecret(32)
		if csrfErr != nil {
			c.JSON(http.StatusOK, APIResponse[any]{Code: BadRequest, Message: "刷新 CSRF Token 失败", Data: nil})
			return
		}
		session.CSRFTokenHash = models.HashSessionSecret(csrfToken)
		if err := models.SaveUserSession(session); err != nil {
			c.JSON(http.StatusOK, APIResponse[any]{Code: BadRequest, Message: "刷新会话失败", Data: nil})
			return
		}
		setCSRFCookie(c, csrfToken, 0)
		csrfCookie = &http.Cookie{Name: csrfCookieName, Value: csrfToken}
	}
	c.JSON(http.StatusOK, APIResponse[any]{
		Code:    Success,
		Message: "获取会话成功",
		Data: gin.H{
			"user":       buildLoginUserResponse(user),
			"session":    buildSessionResponse(session, session.SessionID),
			"csrf_token": csrfCookie.Value,
		},
	})
}

// LogoutAction 清除登录 Cookie
func LogoutAction(c *gin.Context) {
	if user, ok := CurrentUser(c); ok {
		if session, ok := CurrentSession(c); ok {
			_ = models.RevokeUserSession(user.ID, session.SessionID, "logout")
		}
	}
	clearSessionCookies(c)
	c.JSON(http.StatusOK, APIResponse[any]{Code: Success, Message: "退出登录成功", Data: nil})
}

func buildLoginUserResponse(user *models.User) map[string]string {
	return map[string]string{
		"id":       fmt.Sprintf("%d", user.ID),
		"username": user.Username,
		"email":    "",
		"role":     "admin",
	}
}

func buildSessionResponse(session *models.UserSession, currentSessionID string) gin.H {
	return gin.H{
		"session_id":   session.SessionID,
		"current":      session.SessionID == currentSessionID,
		"ip_address":   session.IPAddress,
		"user_agent":   session.UserAgent,
		"last_seen_at": session.LastSeenAt,
		"expires_at":   session.ExpiresAt,
		"created_at":   session.CreatedAt,
	}
}

// ChangePassword 修改密码或用户名
// @Summary 修改密码
// @Description 修改当前登录用户的用户名和密码
// @Tags 用户管理
// @Accept json
// @Produce json
// @Param username body string true "新用户名"
// @Param new_password body string true "新密码"
// @Success 200 {object} object
// @Failure 200 {object} object
// @Router /user/change [post]
// @Security JwtAuth
// @Security ApiKeyAuth
func ChangePassword(c *gin.Context) {
	var req struct {
		Username    string `json:"username" form:"username"`
		NewPassword string `json:"new_password" form:"new_password"`
	}
	if err := c.ShouldBind(&req); err != nil {
		c.JSON(http.StatusOK, APIResponse[any]{Code: BadRequest, Message: fmt.Sprintf("参数错误：%v", err), Data: nil})
		return
	}
	if req.Username == "" {
		c.JSON(http.StatusOK, APIResponse[any]{Code: BadRequest, Message: "用户名不能为空", Data: nil})
		return
	}
	currentUser, ok := CurrentUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, APIResponse[any]{Code: BadRequest, Message: "用户未登录", Data: nil})
		return
	}
	isChange := false
	isChange2 := false
	var err error
	if req.Username != currentUser.Username {
		isChange = true
	}
	isChange2, err = currentUser.ChangeUsernameAndPassword(req.Username, req.NewPassword)
	if err != nil {
		c.JSON(http.StatusOK, APIResponse[any]{Code: BadRequest, Message: "修改失败：" + err.Error(), Data: nil})
		return
	}
	if isChange || isChange2 {
		currentSessionID := ""
		if session, ok := CurrentSession(c); ok {
			currentSessionID = session.SessionID
		}
		_ = models.RevokeOtherUserSessions(currentUser.ID, currentSessionID, "credential_changed")
	}
	c.JSON(http.StatusOK, APIResponse[any]{Code: Success, Message: "修改成功", Data: isChange || isChange2})
}

// GetTwoFactorStatus 获取两步验证状态
func GetTwoFactorStatus(c *gin.Context) {
	currentUser, ok := CurrentUser(c)
	if !ok {
		c.JSON(http.StatusOK, APIResponse[any]{Code: BadRequest, Message: "获取两步验证状态失败", Data: nil})
		return
	}
	user, err := models.GetUserById(currentUser.ID)
	if err != nil {
		c.JSON(http.StatusOK, APIResponse[any]{Code: BadRequest, Message: "获取两步验证状态失败", Data: nil})
		return
	}
	c.JSON(http.StatusOK, APIResponse[any]{Code: Success, Message: "获取两步验证状态成功", Data: gin.H{
		"enabled": user.IsTwoFactorEnabled(),
	}})
}

// SetupTwoFactor 创建两步验证配置草稿
func SetupTwoFactor(c *gin.Context) {
	currentUser, ok := CurrentUser(c)
	if !ok {
		c.JSON(http.StatusOK, APIResponse[any]{Code: BadRequest, Message: "获取用户失败", Data: nil})
		return
	}
	user, err := models.GetUserById(currentUser.ID)
	if err != nil {
		c.JSON(http.StatusOK, APIResponse[any]{Code: BadRequest, Message: "获取用户失败", Data: nil})
		return
	}
	secret, otpURL, err := helpers.GenerateTOTPSecret("QMediaSync", user.Username)
	if err != nil {
		c.JSON(http.StatusOK, APIResponse[any]{Code: BadRequest, Message: "生成两步验证配置失败", Data: nil})
		return
	}
	encryptedSecret, err := helpers.EncryptLocalSecret(secret)
	if err != nil {
		c.JSON(http.StatusOK, APIResponse[any]{Code: BadRequest, Message: "生成两步验证配置失败，请检查配置目录是否可写", Data: nil})
		return
	}
	user.TwoFactorPendingSecret = encryptedSecret
	if err := models.SaveUser(user); err != nil {
		c.JSON(http.StatusOK, APIResponse[any]{Code: BadRequest, Message: "保存两步验证配置失败，请检查数据库状态", Data: nil})
		return
	}
	c.JSON(http.StatusOK, APIResponse[any]{Code: Success, Message: "生成两步验证密钥成功", Data: gin.H{
		"secret":      secret,
		"otpauth_url": otpURL,
	}})
}

// EnableTwoFactor 启用两步验证
func EnableTwoFactor(c *gin.Context) {
	var req EnableTwoFactorRequest
	if err := c.ShouldBind(&req); err != nil || req.TOTPCode == "" {
		c.JSON(http.StatusOK, APIResponse[any]{Code: BadRequest, Message: "验证码错误", Data: nil})
		return
	}
	currentUser, ok := CurrentUser(c)
	if !ok {
		c.JSON(http.StatusOK, APIResponse[any]{Code: BadRequest, Message: "请先生成两步验证密钥", Data: nil})
		return
	}
	user, err := models.GetUserById(currentUser.ID)
	if err != nil || user.TwoFactorPendingSecret == "" {
		c.JSON(http.StatusOK, APIResponse[any]{Code: BadRequest, Message: "请先生成两步验证密钥", Data: nil})
		return
	}
	secret, err := helpers.DecryptLocalSecret(user.TwoFactorPendingSecret)
	if err != nil || !helpers.ValidateTOTPCode(secret, req.TOTPCode) {
		c.JSON(http.StatusOK, APIResponse[any]{Code: BadRequest, Message: "验证码错误", Data: nil})
		return
	}
	user.EnableTwoFactor(user.TwoFactorPendingSecret)
	if err := models.SaveUser(user); err != nil {
		c.JSON(http.StatusOK, APIResponse[any]{Code: BadRequest, Message: "启用两步验证失败", Data: nil})
		return
	}
	currentSessionID := ""
	if session, ok := CurrentSession(c); ok {
		currentSessionID = session.SessionID
	}
	_ = models.RevokeOtherUserSessions(currentUser.ID, currentSessionID, "two_factor_changed")
	c.JSON(http.StatusOK, APIResponse[any]{Code: Success, Message: "启用两步验证成功", Data: nil})
}

// DisableTwoFactor 关闭两步验证
func DisableTwoFactor(c *gin.Context) {
	var req DisableTwoFactorRequest
	if err := c.ShouldBind(&req); err != nil || !req.IsValid() {
		c.JSON(http.StatusOK, APIResponse[any]{Code: BadRequest, Message: "关闭两步验证失败", Data: nil})
		return
	}
	currentUser, ok := CurrentUser(c)
	if !ok {
		c.JSON(http.StatusOK, APIResponse[any]{Code: BadRequest, Message: "关闭两步验证失败", Data: nil})
		return
	}
	user, err := models.GetUserById(currentUser.ID)
	if err != nil || !user.IsTwoFactorEnabled() {
		c.JSON(http.StatusOK, APIResponse[any]{Code: BadRequest, Message: "关闭两步验证失败", Data: nil})
		return
	}
	if _, err := models.CheckLogin(user.Username, req.Password); err != nil {
		c.JSON(http.StatusOK, APIResponse[any]{Code: BadRequest, Message: "关闭两步验证失败", Data: nil})
		return
	}
	secret, err := helpers.DecryptLocalSecret(user.TwoFactorSecret)
	if err != nil || !helpers.ValidateTOTPCode(secret, req.TOTPCode) {
		c.JSON(http.StatusOK, APIResponse[any]{Code: BadRequest, Message: "关闭两步验证失败", Data: nil})
		return
	}
	user.DisableTwoFactor()
	if err := models.SaveUser(user); err != nil {
		c.JSON(http.StatusOK, APIResponse[any]{Code: BadRequest, Message: "关闭两步验证失败", Data: nil})
		return
	}
	currentSessionID := ""
	if session, ok := CurrentSession(c); ok {
		currentSessionID = session.SessionID
	}
	_ = models.RevokeOtherUserSessions(currentUser.ID, currentSessionID, "two_factor_changed")
	c.JSON(http.StatusOK, APIResponse[any]{Code: Success, Message: "关闭两步验证成功", Data: nil})
}

// GetUserInfo 获取当前用户信息
// @Summary 获取用户信息
// @Description 获取当前登录用户的 ID 和用户名
// @Tags 用户管理
// @Accept json
// @Produce json
// @Success 200 {object} object
// @Failure 200 {object} object
// @Router /user/info [get]
// @Security JwtAuth
// @Security ApiKeyAuth
func GetUserInfo(c *gin.Context) {
	currentUser, ok := CurrentUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, APIResponse[any]{Code: BadRequest, Message: "用户未登录", Data: nil})
		return
	}
	// 返回当前用户 ID 和用户名
	respData := make(map[string]string)
	respData["id"] = fmt.Sprintf("%d", currentUser.ID)
	respData["username"] = currentUser.Username
	c.JSON(http.StatusOK, APIResponse[any]{Code: Success, Message: "获取用户信息成功", Data: respData})
}
