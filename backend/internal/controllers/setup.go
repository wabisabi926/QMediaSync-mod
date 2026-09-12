package controllers

import (
	"net/http"

	"qmediasync/internal/models"

	"github.com/gin-gonic/gin"
)

type setupStatusResponse struct {
	Required bool `json:"required"`
}

type createInitialAdminRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// SetupStatusAction 返回是否需要创建首个管理员。
func SetupStatusAction(c *gin.Context) {
	hasUser, err := models.HasAnyUser()
	if err != nil {
		c.JSON(http.StatusOK, APIResponse[any]{Code: BadRequest, Message: "查询初始化状态失败：" + err.Error(), Data: nil})
		return
	}
	c.JSON(http.StatusOK, APIResponse[setupStatusResponse]{
		Code:    Success,
		Message: "查询初始化状态成功",
		Data:    setupStatusResponse{Required: !hasUser},
	})
}

// CreateInitialAdminAction 直接填写用户名和密码创建首个管理员（不使用初始化码）。
func CreateInitialAdminAction(c *gin.Context) {
	var req createInitialAdminRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, APIResponse[any]{Code: BadRequest, Message: "请求参数错误：" + err.Error(), Data: nil})
		return
	}
	user, err := models.CreateInitialAdmin(req.Username, req.Password)
	if err != nil {
		c.JSON(http.StatusOK, APIResponse[any]{Code: BadRequest, Message: "创建管理员失败：" + err.Error(), Data: nil})
		return
	}
	c.JSON(http.StatusOK, APIResponse[gin.H]{
		Code:    Success,
		Message: "管理员创建成功，请使用新账号登录",
		Data:    gin.H{"username": user.Username},
	})
}