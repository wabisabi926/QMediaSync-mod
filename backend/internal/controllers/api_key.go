package controllers

import (
	"fmt"
	"net/http"
	"strconv"

	"Q115-STRM/internal/models"

	"github.com/gin-gonic/gin"
)

// CreateAPIKeyRequest 创建 API Key 请求。
type CreateAPIKeyRequest struct {
	Name string `json:"name" binding:"required"`
}

// CreateAPIKeyResponse 创建 API Key 响应。
type CreateAPIKeyResponse struct {
	ID        uint   `json:"id"`
	Name      string `json:"name"`
	Key       string `json:"key"`        // 完整的 API Key，仅此一次返回
	KeyPrefix string `json:"key_prefix"` // 前缀用于显示
	CreatedAt int64  `json:"created_at"`
	IsActive  bool   `json:"is_active"`
}

// APIKeyListItem API Key 列表项（不包含完整密钥）。
type APIKeyListItem struct {
	ID         uint   `json:"id"`
	Name       string `json:"name"`
	KeyPrefix  string `json:"key_prefix"`
	LastUsedAt int64  `json:"last_used_at"`
	CreatedAt  int64  `json:"created_at"`
	IsActive   bool   `json:"is_active"`
}

// CreateAPIKey 创建新的 API Key。
// @Summary 创建 API Key
// @Description 为当前登录用户创建新的 API Key，仅此一次返回完整密钥
// @Tags API 管理
// @Accept json
// @Produce json
// @Param name body string true "API Key 名称"
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

	currentUser, ok := CurrentUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, APIResponse[any]{Code: BadRequest, Message: "用户未登录", Data: nil})
		return
	}

	// 创建 API Key
	apiKey, rawKey, err := models.CreateAPIKey(currentUser.ID, req.Name)
	if err != nil {
		c.JSON(http.StatusOK, APIResponse[any]{Code: BadRequest, Message: fmt.Sprintf("创建 API Key 失败：%v", err), Data: nil})
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
		Message: "API Key 已创建，请妥善保管密钥，此密钥仅显示一次",
		Data:    resp,
	})
}

// ListAPIKeys 获取 API Key 列表。
// @Summary 获取 API Key 列表
// @Description 获取当前登录用户的所有 API Key（不包含完整密钥）
// @Tags API 管理
// @Accept json
// @Produce json
// @Success 200 {object} object
// @Failure 200 {object} object
// @Router /api-key/list [get]
// @Security JwtAuth
// @Security ApiKeyAuth
func ListAPIKeys(c *gin.Context) {
	currentUser, ok := CurrentUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, APIResponse[any]{Code: BadRequest, Message: "用户未登录", Data: nil})
		return
	}

	// 查询用户的 API Key
	apiKeys, err := models.GetAPIKeysByUserID(currentUser.ID)
	if err != nil {
		c.JSON(http.StatusOK, APIResponse[any]{Code: BadRequest, Message: fmt.Sprintf("查询 API Key 失败：%v", err), Data: nil})
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

// DeleteAPIKey 删除 API Key。
// @Summary 删除 API Key
// @Description 删除指定 ID 的 API Key（仅能删除自己的）
// @Tags API 管理
// @Accept json
// @Produce json
// @Param id path integer true "API Key ID"
// @Success 200 {object} object
// @Failure 200 {object} object
// @Router /api-key/delete/{id} [delete]
// @Security JwtAuth
// @Security ApiKeyAuth
func DeleteAPIKey(c *gin.Context) {
	currentUser, ok := CurrentUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, APIResponse[any]{Code: BadRequest, Message: "用户未登录", Data: nil})
		return
	}

	// 获取 API Key ID
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusOK, APIResponse[any]{Code: BadRequest, Message: "无效的 API Key ID", Data: nil})
		return
	}

	// 删除 API Key（确保只能删除自己的）
	err = models.DeleteAPIKey(uint(id), currentUser.ID)
	if err != nil {
		c.JSON(http.StatusOK, APIResponse[any]{Code: BadRequest, Message: fmt.Sprintf("删除 API Key 失败：%v", err), Data: nil})
		return
	}

	c.JSON(http.StatusOK, APIResponse[any]{
		Code:    Success,
		Message: "删除成功",
		Data:    nil,
	})
}

// UpdateAPIKeyStatusRequest 更新 API Key 状态请求。
type UpdateAPIKeyStatusRequest struct {
	IsActive bool `json:"is_active"`
}

// UpdateAPIKeyStatus 更新 API Key 状态。
// @Summary 启用或禁用 API Key
// @Description 更新指定 API Key 的启用状态
// @Tags API 管理
// @Accept json
// @Produce json
// @Param id path integer true "API Key ID"
// @Param is_active body boolean true "是否启用"
// @Success 200 {object} object
// @Failure 200 {object} object
// @Router /api-key/status/{id} [put]
// @Security JwtAuth
// @Security ApiKeyAuth
func UpdateAPIKeyStatus(c *gin.Context) {
	currentUser, ok := CurrentUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, APIResponse[any]{Code: BadRequest, Message: "用户未登录", Data: nil})
		return
	}

	// 获取 API Key ID
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusOK, APIResponse[any]{Code: BadRequest, Message: "无效的 API Key ID", Data: nil})
		return
	}

	// 解析请求体
	var req UpdateAPIKeyStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, APIResponse[any]{Code: BadRequest, Message: fmt.Sprintf("参数错误：%v", err), Data: nil})
		return
	}

	// 更新 API Key 状态（确保只能更新自己的）
	err = models.UpdateAPIKeyStatus(uint(id), currentUser.ID, req.IsActive)
	if err != nil {
		c.JSON(http.StatusOK, APIResponse[any]{Code: BadRequest, Message: fmt.Sprintf("更新 API Key 状态失败：%v", err), Data: nil})
		return
	}

	statusText := "禁用"
	if req.IsActive {
		statusText = "启用"
	}

	c.JSON(http.StatusOK, APIResponse[any]{
		Code:    Success,
		Message: fmt.Sprintf("API Key 已%s", statusText),
		Data:    nil,
	})
}
