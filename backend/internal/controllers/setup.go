package controllers

import (
	"net/http"
	"sync"

	"qmediasync/internal/models"

	"github.com/gin-gonic/gin"
)

type initialSetupState struct {
	sync.Mutex
	enabled bool
}

var initialSetup initialSetupState

type setupStatusResponse struct {
	Required bool `json:"required"`
}

type createInitialAdminRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// ConfigureInitialSetup 根据是否需要初始化管理员启用或关闭初始化入口。
func ConfigureInitialSetup(required bool) (string, error) {
	if !required {
		DisableInitialSetup()
		return "", nil
	}
	initialSetup.Lock()
	defer initialSetup.Unlock()
	initialSetup.enabled = true
	return "", nil
}

// DisableInitialSetup 关闭管理员初始化入口。
func DisableInitialSetup() {
	initialSetup.Lock()
	defer initialSetup.Unlock()
	initialSetup.enabled = false
}

func isInitialSetupRequired() bool {
	initialSetup.Lock()
	defer initialSetup.Unlock()
	return initialSetup.enabled
}

// SetupStatusAction 返回是否需要创建首个管理员。
func SetupStatusAction(c *gin.Context) {
	hasUser, err := models.HasAnyUser()
	if err != nil {
		c.JSON(http.StatusOK, APIResponse[any]{Code: BadRequest, Message: "查询初始化状态失败：" + err.Error(), Data: nil})
		return
	}
	required := !hasUser && isInitialSetupRequired()
	c.JSON(http.StatusOK, APIResponse[setupStatusResponse]{
		Code:    Success,
		Message: "查询初始化状态成功",
		Data:    setupStatusResponse{Required: required},
	})
}

// CreateInitialAdminAction 创建首个管理员（无需初始化码）。
func CreateInitialAdminAction(c *gin.Context) {
	var req createInitialAdminRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, APIResponse[any]{Code: BadRequest, Message: "请求参数错误：" + err.Error(), Data: nil})
		return
	}
	user, err := models.CreateInitialAdmin(req.Username, req.Password)
	if err != nil {
		if err == models.ErrInitialAdminAlreadyExists {
			DisableInitialSetup()
			c.JSON(http.StatusOK, APIResponse[any]{Code: BadRequest, Message: "管理员已初始化", Data: nil})
			return
		}
		c.JSON(http.StatusOK, APIResponse[any]{Code: BadRequest, Message: "创建管理员失败：" + err.Error(), Data: nil})
		return
	}
	DisableInitialSetup()
	c.JSON(http.StatusOK, APIResponse[gin.H]{
		Code:    Success,
		Message: "管理员创建成功，请使用新账号登录",
		Data:    gin.H{"username": user.Username},
	})
}
