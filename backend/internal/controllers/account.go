package controllers

import (
	"fmt"
	"net/http"
	"strings"

	"Q115-STRM/internal/helpers"
	"Q115-STRM/internal/models"

	"github.com/gin-gonic/gin"
)

// GetAccountList 获取所有开放平台账号列表
// @Summary 获取账号列表
// @Description 获取所有已配置的开放平台账号（如115、OpenList等）
// @Tags 账号管理
// @Accept json
// @Produce json
// @Success 200 {object} object
// @Failure 200 {object} object
// @Router /account/list [get]
// @Security JwtAuth
// @Security ApiKeyAuth
func GetAccountList(c *gin.Context) {
	accounts, err := models.GetAllAccount()
	if err != nil {
		c.JSON(http.StatusOK, APIResponse[any]{Code: BadRequest, Message: "查询开放平台账号失败", Data: nil})
		return
	}
	type accountResp struct {
		ID                uint              `json:"id"`
		SourceType        models.SourceType `json:"source_type"`
		Name              string            `json:"name"`
		AppId             string            `json:"app_id"`
		AppName           string            `json:"app_id_name"`
		Username          string            `json:"username"`
		UserId            string            `json:"user_id"`
		Token             string            `json:"token"`
		CreatedAt         int64             `json:"created_at"`
		TokenFailedReason string            `json:"token_failed_reason"`
		BaseUrl           string            `json:"base_url"`
		AuthType          string            `json:"auth_type"`
	}
	resp := make([]accountResp, 0, len(accounts))
	for _, account := range accounts {
		a := accountResp{
			ID:                account.ID,
			SourceType:        account.SourceType,
			Name:              account.Name,
			AppId:             "",
			AppName:           "",
			Username:          account.Username,
			UserId:            string(account.UserId),
			Token:             account.Token,
			CreatedAt:         account.CreatedAt,
			TokenFailedReason: account.TokenFailedReason,
			BaseUrl:           account.BaseUrl,
		}
		switch account.AppId {
		case models.BuiltIn115AppQ115STRM:
			a.AppId = ""
			a.AppName = models.BuiltIn115AppQ115STRM
		case models.BuiltIn115AppMQMediaLibrary:
			a.AppId = ""
			a.AppName = models.BuiltIn115AppMQMediaLibrary
		case models.BuiltIn115AppQMediaSync:
			a.AppId = ""
			a.AppName = models.BuiltIn115AppQMediaSync
		default:
			a.AppName = strings.TrimSpace(account.AppIdName)
			if a.AppName == "" {
				a.AppName = models.Custom115AppName
			}
			a.AppId = account.AppId
		}
		if a.Name == "" {
			a.Name = account.Username
		}
		if account.SourceType == models.SourceTypeOpenList {
			if account.Password != "" {
				a.AuthType = "password"
			} else {
				a.AuthType = "token"
			}
		}
		resp = append(resp, a)
	}

	c.JSON(http.StatusOK, APIResponse[[]accountResp]{Code: Success, Message: "查询开放平台账号成功", Data: resp})
}

// CreateTmpAccount 创建临时开放平台账号
// @Summary 创建账号
// @Description 创建新的开放平台账号（支持115、OpenList等）
// @Tags 账号管理
// @Accept json
// @Produce json
// @Param source_type query string true "账号源类型"
// @Param name query string true "账号名称"
// @Param app_id query string false "应用ID（自定义时必需）"
// @Param app_id_name query string false "选择的115开放平台应用（QMediaSync、Q115-STRM、MQ的媒体库、自定义）"
// @Success 200 {object} object
// @Failure 200 {object} object
// @Router /account/create [post]
// @Security JwtAuth
// @Security ApiKeyAuth
func CreateTmpAccount(c *gin.Context) {
	type tmpAccountReq struct {
		SourceType    models.SourceType `json:"source_type" form:"source_type"`
		Name          string            `json:"name" form:"name"`
		AppId         string            `json:"app_id" form:"app_id"`
		SelectedApp   string            `json:"app_id_name" form:"app_id_name"`
		CustomAppName string            `json:"custom_app_name" form:"custom_app_name"`
	}
	tmpAccount := &tmpAccountReq{}
	if err := c.ShouldBind(tmpAccount); err != nil {
		c.JSON(http.StatusBadRequest, APIResponse[any]{Code: BadRequest, Message: "请求参数错误", Data: nil})
		return
	}
	// 创建临时账号
	var appId string
	var appIdName string
	if tmpAccount.SourceType == models.SourceTypeBaiduPan {
		appId = helpers.GlobalConfig.BaiDuPanAppId
	}

	if models.SourceType115 == tmpAccount.SourceType {
		switch tmpAccount.SelectedApp {
		case models.BuiltIn115AppQ115STRM, models.BuiltIn115AppMQMediaLibrary, models.BuiltIn115AppQMediaSync:
			appId = tmpAccount.SelectedApp
		case models.Custom115AppName:
			appId = strings.TrimSpace(tmpAccount.AppId)
			appIdName = strings.TrimSpace(tmpAccount.CustomAppName)
			if appId == "" {
				c.JSON(http.StatusOK, APIResponse[any]{Code: BadRequest, Message: "自定义应用ID不能为空", Data: nil})
				return
			}
		default:
			c.JSON(http.StatusOK, APIResponse[any]{Code: BadRequest, Message: "不支持的115开放平台应用", Data: nil})
			return
		}
	}
	account, err := models.CreateAccountByName(
		strings.TrimSpace(tmpAccount.Name),
		tmpAccount.SourceType,
		appId,
		appIdName,
	)
	if err != nil {
		c.JSON(http.StatusOK, APIResponse[any]{Code: BadRequest, Message: "创建开放平台账号失败", Data: nil})
		return
	}
	c.JSON(http.StatusOK, APIResponse[models.Account]{Code: Success, Message: "创建开放平台账号成功", Data: *account})
}

// UpdateAccountInfo 更新账号资料
// @Summary 更新账号资料
// @Description 更新账号备注和自定义开放平台应用名，不修改授权凭据或OpenList连接配置
// @Tags 账号管理
// @Accept json
// @Produce json
// @Param id query integer true "账号ID"
// @Param name query string false "账号备注"
// @Param app_id_name query string false "自定义应用名"
// @Success 200 {object} object
// @Failure 200 {object} object
// @Router /account/update [post]
// @Security JwtAuth
// @Security ApiKeyAuth
func UpdateAccountInfo(c *gin.Context) {
	type updateAccountInfoReq struct {
		ID            uint   `json:"id" form:"id"`
		Name          string `json:"name" form:"name"`
		CustomAppName string `json:"app_id_name" form:"app_id_name"`
	}
	req := &updateAccountInfoReq{}
	if err := c.ShouldBind(req); err != nil || req.ID == 0 {
		c.JSON(http.StatusBadRequest, APIResponse[any]{Code: BadRequest, Message: "请求参数错误", Data: nil})
		return
	}
	account, err := models.GetAccountById(req.ID)
	if err != nil {
		c.JSON(http.StatusOK, APIResponse[any]{Code: BadRequest, Message: "查询开放平台账号失败", Data: nil})
		return
	}
	if err := account.UpdateInfo(strings.TrimSpace(req.Name), strings.TrimSpace(req.CustomAppName)); err != nil {
		c.JSON(http.StatusOK, APIResponse[any]{Code: BadRequest, Message: "更新开放平台账号资料失败", Data: nil})
		return
	}
	c.JSON(http.StatusOK, APIResponse[any]{Code: Success, Message: "更新开放平台账号资料成功", Data: nil})
}

// DeleteAccount 删除开放平台账号
// @Summary 删除账号
// @Description 删除指定ID的开放平台账号
// @Tags 账号管理
// @Accept json
// @Produce json
// @Param id query integer true "账号ID"
// @Success 200 {object} object
// @Failure 200 {object} object
// @Router /account/delete [delete]
// @Security JwtAuth
// @Security ApiKeyAuth
func DeleteAccount(c *gin.Context) {
	type deleteAccountReq struct {
		ID uint `json:"id" form:"id"`
	}
	req := &deleteAccountReq{}
	if err := c.ShouldBind(req); err != nil {
		c.JSON(http.StatusBadRequest, APIResponse[any]{Code: BadRequest, Message: "请求参数错误", Data: nil})
		return
	}
	account, err := models.GetAccountById(req.ID)
	if err != nil {
		c.JSON(http.StatusOK, APIResponse[any]{Code: BadRequest, Message: "查询开放平台账号失败", Data: nil})
		return
	}
	err = account.Delete()
	if err != nil {
		c.JSON(http.StatusOK, APIResponse[any]{Code: BadRequest, Message: err.Error(), Data: nil})
		return
	}
	c.JSON(http.StatusOK, APIResponse[any]{Code: Success, Message: "删除开放平台账号成功", Data: nil})
}

// CreateOpenListAccount 创建或更新OpenList账号
// @Summary 创建/更新OpenList账号
// @Description 创建新的OpenList账号或更新现有账号的凭证，支持直接使用Token认证
// @Tags 账号管理
// @Accept json
// @Produce json
// @Param id query integer false "账号ID（指定则为更新操作）"
// @Param base_url query string true "OpenList服务器地址"
// @Param username query string true "用户名"
// @Param password query string true "密码"
// @Param token query string false "直接提供的访问Token（优先使用）"
// @Success 200 {object} object
// @Failure 200 {object} object
// @Router /account/openlist [post]
// @Security JwtAuth
// @Security ApiKeyAuth
func CreateOpenListAccount(c *gin.Context) {
	type createOpenListAccountReq struct {
		Id       uint   `json:"id" form:"id"`
		BaseUrl  string `json:"base_url" form:"base_url"`
		Username string `json:"username" form:"username"`
		Password string `json:"password" form:"password"`
		Token    string `json:"token" form:"token"`
	}
	req := &createOpenListAccountReq{}
	if err := c.ShouldBind(req); err != nil {
		c.JSON(http.StatusBadRequest, APIResponse[any]{Code: BadRequest, Message: "请求参数错误", Data: nil})
		return
	}
	if req.BaseUrl == "" {
		c.JSON(http.StatusBadRequest, APIResponse[any]{Code: BadRequest, Message: "请求参数错误", Data: nil})
		return
	}
	if req.Token == "" && (req.Password == "" || req.Username == "") {
		c.JSON(http.StatusBadRequest, APIResponse[any]{Code: BadRequest, Message: "必须提供Token或用户名密码", Data: nil})
		return
	}
	// 如果不以http开头则添加http://
	if !strings.HasPrefix(req.BaseUrl, "http://") && !strings.HasPrefix(req.BaseUrl, "https://") {
		req.BaseUrl = "http://" + req.BaseUrl
	}
	// 如果结尾有/则删除
	req.BaseUrl = strings.TrimSuffix(req.BaseUrl, "/")
	if req.Id != 0 {
		account, err := models.GetAccountById(req.Id)
		if err != nil {
			c.JSON(http.StatusOK, APIResponse[any]{Code: BadRequest, Message: fmt.Sprintf("查询openlist账号失败: %s", err.Error()), Data: nil})
			return
		}
		if err := account.UpdateOpenList(req.BaseUrl, req.Username, req.Password, req.Token); err != nil {
			c.JSON(http.StatusOK, APIResponse[any]{Code: BadRequest, Message: fmt.Sprintf("更新openlist账号失败: %s", err.Error()), Data: nil})
			return
		}
		c.JSON(http.StatusOK, APIResponse[any]{Code: Success, Message: "更新openlist账号成功", Data: nil})
		return
	}
	// 创建openlist账号
	_, err := models.CreateOpenListAccount(req.BaseUrl, req.Username, req.Password, req.Token)
	if err != nil {
		c.JSON(http.StatusOK, APIResponse[any]{Code: BadRequest, Message: fmt.Sprintf("创建openlist账号失败: %s", err.Error()), Data: nil})
		return
	}
	c.JSON(http.StatusOK, APIResponse[any]{Code: Success, Message: "创建openlist账号成功", Data: nil})
}
