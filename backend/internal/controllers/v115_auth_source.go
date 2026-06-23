package controllers

import (
	"Q115-STRM/internal/v115auth"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// GetV115AppIDSources 查询 115 Open APPID 目录。
func GetV115AppIDSources(c *gin.Context) {
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	result := v115auth.SearchBuiltInAppIDSources(c.Query("keyword"), offset, limit)
	c.JSON(http.StatusOK, APIResponse[v115auth.AppIDSearchResult]{Code: Success, Message: "查询115 APPID目录成功", Data: result})
}
