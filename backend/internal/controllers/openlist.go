package controllers

import (
	"Q115-STRM/internal/helpers"
	"Q115-STRM/internal/models"
	"net/http"

	"github.com/gin-gonic/gin"
)

// GetOpenListFileUrl 获取OpenList文件直链
// @Summary 获取OpenList文件直链
// @Description 根据路径查询OpenList文件直链并302重定向
// @Tags OpenList
// @Accept json
// @Produce json
// @Param account_id query integer true "账号ID"
// @Param path query string true "文件路径"
// @Success 302 {string} string "重定向到文件直链"
// @Failure 200 {object} object
// @Router /openlist/url [get]
func GetOpenListFileUrl(c *gin.Context) {
	type fileUrlReq struct {
		AccountId uint   `json:"account_id" form:"account_id"`
		Path      string `json:"path" form:"path"`
	}
	var req fileUrlReq
	if err := c.ShouldBind(&req); err != nil {
		c.JSON(http.StatusBadRequest, APIResponse[any]{Code: BadRequest, Message: "参数错误", Data: nil})
		return
	}
	if req.Path == "" {
		c.JSON(http.StatusBadRequest, APIResponse[any]{Code: BadRequest, Message: "path 参数不能为空", Data: nil})
		return
	}
	account, err := models.GetAccountById(req.AccountId)
	if err != nil {
		c.JSON(http.StatusBadRequest, APIResponse[any]{Code: BadRequest, Message: "账号ID不存在", Data: nil})
		return
	}
	client := account.GetOpenListClient()
	fileDetail, err := client.FileDetail(req.Path)
	if err != nil {
		helpers.AppLogger.Errorf("[下载] 获取文件详情失败: %v", err)
		c.JSON(http.StatusOK, APIResponse[any]{Code: BadRequest, Message: "获取文件详情失败", Data: nil})
		return
	}
	if fileDetail.RawURL == "" {
		c.JSON(http.StatusOK, APIResponse[any]{Code: BadRequest, Message: "文件详情中未找到直链", Data: nil})
		return
	}
	// 302跳转到直链
	c.Redirect(http.StatusFound, fileDetail.RawURL)
}
