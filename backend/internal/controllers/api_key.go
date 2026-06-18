package controllers

import (
	"Q115-STRM/internal/models"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// CreateAPIKeyRequest 创建API Key请求
type CreateAPIKeyRequest struct {
	Name string `json:"name" binding:"required"`
}

// CreateAPIKeyResponse 创建API Key响应
type CreateAPIKeyResponse struct {
	ID        uint   `json:"id"`
	Name      string `json:"name"`
	Key       string `json:"key"`        // 完整的API Key，仅此一次返回
	KeyPrefix string `json:"key_prefix"` // 前缀用于显示
	CreatedAt int64  `json:"created_at"`
	IsActive  bool   `json:"is_active"`
}

// APIKeyListItem API Key列表项（不包含完整密钥）
type APIKeyListItem struct {
	ID         uint   `json:"id"`
	Name       string `json:"name"`
	KeyPrefix  string `json:"key_prefix"`
	LastUsedAt int64  `json:"last_used_at"`
	CreatedAt  int64  `json:"created_at"`
	IsActive   bool   `json:"is_active"`
}

// CreateAPIKey 创建新的API Key
// @Summary 创建API密钥
// @Description 为当前登录用户创建新的API密钥，仅此一次返回完整密钥
// @Tags API管理
// @Accept json
// @Produce json
// @Param name body string true "API密钥名称"
// @Success 200 {object} object
// @Failure 200 {object} object
// @Router /api-key/create [post]
// @Security JwtAuth
// @Security ApiKeyAuth
func CreateAPIKey(c *gin.Context) {
	var req CreateAPIKeyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, APIResponse[any]{Code: BadRequest, Message: fmt.Sprintf("参数错误：%v", err), Data: nil})
		return
	}

	// 获取当前登录用户
	if LoginedUser == nil {
		c.JSON(http.StatusUnauthorized, APIResponse[any]{Code: BadRequest, Message: "用户未登录", Data: nil})
		return
	}

	// 创建API Key
	apiKey, rawKey, err := models.CreateAPIKey(LoginedUser.ID, req.Name)
	if err != nil {
		c.JSON(http.StatusOK, APIResponse[any]{Code: BadRequest, Message: fmt.Sprintf("创建API Key失败：%v", err), Data: nil})
		return
	}

	// 返回包含完整密钥的响应（仅此一次）
	resp := CreateAPIKeyResponse{
		ID:        apiKey.ID,
		Name:      apiKey.Name,
		Key:       rawKey, // 完整密钥仅返回一次
		KeyPrefix: apiKey.KeyPrefix,
		CreatedAt: apiKey.CreatedAt,
		IsActive:  apiKey.IsActive,
	}

	c.JSON(http.StatusOK, APIResponse[CreateAPIKeyResponse]{
		Code:    Success,
		Message: "API Key创建成功，请妥善保管密钥，此密钥仅显示一次",
		Data:    resp,
	})
}

// ListAPIKeys 获取API密钥列表
// @Summary 获取API密钥列表
// @Description 获取当前登录用户的所有API密钥（不包含完整密钥）
// @Tags API管理
// @Accept json
// @Produce json
// @Success 200 {object} object
// @Failure 200 {object} object
// @Router /api-key/list [get]
// @Security JwtAuth
// @Security ApiKeyAuth
func ListAPIKeys(c *gin.Context) {
	// 获取当前登录用户
	if LoginedUser == nil {
		c.JSON(http.StatusUnauthorized, APIResponse[any]{Code: BadRequest, Message: "用户未登录", Data: nil})
		return
	}

	// 查询用户的API Keys
	apiKeys, err := models.GetAPIKeysByUserID(LoginedUser.ID)
	if err != nil {
		c.JSON(http.StatusOK, APIResponse[any]{Code: BadRequest, Message: fmt.Sprintf("查询API Keys失败：%v", err), Data: nil})
		return
	}

	// 转换为响应格式（不包含完整密钥）
	resp := make([]APIKeyListItem, 0, len(apiKeys))
	for _, apiKey := range apiKeys {
		resp = append(resp, APIKeyListItem{
			ID:         apiKey.ID,
			Name:       apiKey.Name,
			KeyPrefix:  apiKey.KeyPrefix,
			LastUsedAt: apiKey.LastUsedAt,
			CreatedAt:  apiKey.CreatedAt,
			IsActive:   apiKey.IsActive,
		})
	}

	c.JSON(http.StatusOK, APIResponse[[]APIKeyListItem]{
		Code:    Success,
		Message: "查询成功",
		Data:    resp,
	})
}

// DeleteAPIKey 删除API密钥
// @Summary 删除API密钥
// @Description 删除指定ID的API密钥（仅能删除自己的）
// @Tags API管理
// @Accept json
// @Produce json
// @Param id path integer true "API密钥ID"
// @Success 200 {object} object
// @Failure 200 {object} object
// @Router /api-key/delete/{id} [delete]
// @Security JwtAuth
// @Security ApiKeyAuth
func DeleteAPIKey(c *gin.Context) {
	// 获取当前登录用户
	if LoginedUser == nil {
		c.JSON(http.StatusUnauthorized, APIResponse[any]{Code: BadRequest, Message: "用户未登录", Data: nil})
		return
	}

	// 获取API Key ID
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusOK, APIResponse[any]{Code: BadRequest, Message: "无效的API Key ID", Data: nil})
		return
	}

	// 删除API Key（确保只能删除自己的）
	err = models.DeleteAPIKey(uint(id), LoginedUser.ID)
	if err != nil {
		c.JSON(http.StatusOK, APIResponse[any]{Code: BadRequest, Message: fmt.Sprintf("删除API Key失败：%v", err), Data: nil})
		return
	}

	c.JSON(http.StatusOK, APIResponse[any]{
		Code:    Success,
		Message: "删除成功",
		Data:    nil,
	})
}

// UpdateAPIKeyStatusRequest 更新API Key状态请求
type UpdateAPIKeyStatusRequest struct {
	IsActive bool `json:"is_active"`
}

// UpdateAPIKeyStatus 更新API密钥状态
// @Summary 启用/禁用API密钥
// @Description 更新指定API密钥的启用/禁用状态
// @Tags API管理
// @Accept json
// @Produce json
// @Param id path integer true "API密钥ID"
// @Param is_active body boolean true "是否启用"
// @Success 200 {object} object
// @Failure 200 {object} object
// @Router /api-key/status/{id} [put]
// @Security JwtAuth
// @Security ApiKeyAuth
func UpdateAPIKeyStatus(c *gin.Context) {
	// 获取当前登录用户
	if LoginedUser == nil {
		c.JSON(http.StatusUnauthorized, APIResponse[any]{Code: BadRequest, Message: "用户未登录", Data: nil})
		return
	}

	// 获取API Key ID
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusOK, APIResponse[any]{Code: BadRequest, Message: "无效的API Key ID", Data: nil})
		return
	}

	// 解析请求体
	var req UpdateAPIKeyStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, APIResponse[any]{Code: BadRequest, Message: fmt.Sprintf("参数错误：%v", err), Data: nil})
		return
	}

	// 更新API Key状态（确保只能更新自己的）
	err = models.UpdateAPIKeyStatus(uint(id), LoginedUser.ID, req.IsActive)
	if err != nil {
		c.JSON(http.StatusOK, APIResponse[any]{Code: BadRequest, Message: fmt.Sprintf("更新API Key状态失败：%v", err), Data: nil})
		return
	}

	statusText := "禁用"
	if req.IsActive {
		statusText = "启用"
	}

	c.JSON(http.StatusOK, APIResponse[any]{
		Code:    Success,
		Message: fmt.Sprintf("API Key已%s", statusText),
		Data:    nil,
	})
}
