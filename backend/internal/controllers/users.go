package controllers

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"Q115-STRM/internal/helpers"
	"Q115-STRM/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

type LoginRequest struct {
	Username   string `json:"username" form:"username"`
	Password   string `json:"password" form:"password"`
	TOTPCode   string `json:"totp_code" form:"totp_code"`
	RememberMe bool   `json:"rememberMe" form:"rememberMe"`
}

var LoginedUser *models.User = nil

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
	user := &models.User{}
	var req LoginRequest
	if err := c.ShouldBind(&req); err != nil {
		writeLoginFailure(c)
		return
	}
	username := req.Username
	password := req.Password
	if username == "" || password == "" {
		writeLoginFailure(c)
		return
	}

	// 查询用户是否存在
	user, userErr := models.CheckLogin(username, password)
	if userErr != nil || user.ID == 0 {
		writeLoginFailure(c)
		return
	}

	if user.IsTwoFactorEnabled() {
		secret, err := helpers.DecryptLocalSecret(user.TwoFactorSecret)
		if err != nil || !helpers.ValidateTOTPCode(secret, req.TOTPCode) {
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

	claims := &LoginUser{
		ID:       user.ID,
		Username: user.Username,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(tokenExpire)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(helpers.GlobalConfig.JwtSecret))
	if err != nil {
		helpers.AppLogger.Errorf("登录签名失败：%v", err)
		writeLoginFailure(c)
		return
	}

	// 设置 HttpOnly Cookie
	c.SetCookie(
		"auth_token", // Cookie 名称
		tokenString,  // Cookie 值
		cookieMaxAge, // MaxAge（秒），0 表示会话 Cookie
		"/",          // Path
		"",           // Domain（空表示当前域名）
		false,        // Secure（false 兼容飞牛 HTTP 环境）
		true,         // HttpOnly（防止 XSS 攻击）
	)
	LoginedUser = user
	res := make(map[string]interface{})
	u := make(map[string]string)
	u["id"] = fmt.Sprintf("%d", user.ID)
	u["username"] = user.Username
	u["email"] = ""
	u["role"] = "admin"
	res["user"] = u
	res["token"] = tokenString
	c.JSON(http.StatusOK, APIResponse[any]{Code: Success, Message: "登录成功", Data: res})
}

func getTokenFromRequest(c *gin.Context) string {
	if cookie, err := c.Request.Cookie("auth_token"); err == nil && cookie.Value != "" {
		return cookie.Value
	}
	authHeader := c.Request.Header.Get("Authorization")
	if strings.HasPrefix(authHeader, "Bearer ") {
		return strings.TrimPrefix(authHeader, "Bearer ")
	}
	return ""
}

// SessionAction 返回当前登录会话
func SessionAction(c *gin.Context) {
	tokenString := getTokenFromRequest(c)
	if tokenString == "" {
		c.JSON(http.StatusOK, APIResponse[any]{Code: BadRequest, Message: "登录凭证不存在", Data: nil})
		return
	}
	loginUser, err := ValidateJWT(tokenString)
	if err != nil {
		c.JSON(http.StatusOK, APIResponse[any]{Code: BadRequest, Message: fmt.Sprintf("登录凭证无效：%v", err), Data: nil})
		return
	}
	user, err := models.GetUserById(loginUser.ID)
	if err != nil {
		c.JSON(http.StatusOK, APIResponse[any]{Code: BadRequest, Message: fmt.Sprintf("获取用户信息失败：%v", err), Data: nil})
		return
	}
	c.JSON(http.StatusOK, APIResponse[any]{
		Code:    Success,
		Message: "获取会话成功",
		Data: map[string]any{
			"token": tokenString,
			"user": map[string]string{
				"id":       fmt.Sprintf("%d", user.ID),
				"username": user.Username,
				"email":    "",
				"role":     "admin",
			},
		},
	})
}

// LogoutAction 清除登录 Cookie
func LogoutAction(c *gin.Context) {
	c.SetCookie("auth_token", "", -1, "/", "", false, true)
	LoginedUser = nil
	c.JSON(http.StatusOK, APIResponse[any]{Code: Success, Message: "退出登录成功", Data: nil})
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
	isChange := false
	isChange2 := false
	var err error
	if req.Username != LoginedUser.Username {
		isChange = true
	}
	isChange2, err = LoginedUser.ChangeUsernameAndPassword(req.Username, req.NewPassword)
	if err != nil {
		c.JSON(http.StatusOK, APIResponse[any]{Code: BadRequest, Message: "修改失败：" + err.Error(), Data: nil})
		return
	}
	c.JSON(http.StatusOK, APIResponse[any]{Code: Success, Message: "修改成功", Data: isChange || isChange2})
}

// GetTwoFactorStatus 获取两步验证状态
func GetTwoFactorStatus(c *gin.Context) {
	if LoginedUser == nil {
		c.JSON(http.StatusOK, APIResponse[any]{Code: BadRequest, Message: "获取两步验证状态失败", Data: nil})
		return
	}
	user, err := models.GetUserById(LoginedUser.ID)
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
	if LoginedUser == nil {
		c.JSON(http.StatusOK, APIResponse[any]{Code: BadRequest, Message: "获取用户失败", Data: nil})
		return
	}
	user, err := models.GetUserById(LoginedUser.ID)
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
	if LoginedUser == nil {
		c.JSON(http.StatusOK, APIResponse[any]{Code: BadRequest, Message: "请先生成两步验证密钥", Data: nil})
		return
	}
	user, err := models.GetUserById(LoginedUser.ID)
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
	c.JSON(http.StatusOK, APIResponse[any]{Code: Success, Message: "启用两步验证成功", Data: nil})
}

// DisableTwoFactor 关闭两步验证
func DisableTwoFactor(c *gin.Context) {
	var req DisableTwoFactorRequest
	if err := c.ShouldBind(&req); err != nil || !req.IsValid() {
		c.JSON(http.StatusOK, APIResponse[any]{Code: BadRequest, Message: "关闭两步验证失败", Data: nil})
		return
	}
	if LoginedUser == nil {
		c.JSON(http.StatusOK, APIResponse[any]{Code: BadRequest, Message: "关闭两步验证失败", Data: nil})
		return
	}
	user, err := models.GetUserById(LoginedUser.ID)
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
	// 返回当前用户 ID 和用户名
	respData := make(map[string]string)
	respData["id"] = fmt.Sprintf("%d", LoginedUser.ID)
	respData["username"] = LoginedUser.Username
	c.JSON(http.StatusOK, APIResponse[any]{Code: Success, Message: "获取用户信息成功", Data: respData})
}
