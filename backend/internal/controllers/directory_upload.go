package controllers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"qmediasync/internal/db"
	"qmediasync/internal/directoryupload"
	"qmediasync/internal/models"

	"github.com/gin-gonic/gin"
)

type directoryUploadRuleRequest struct {
	SyncPathID               uint                                `json:"sync_path_id"`
	AccountID                uint                                `json:"account_id"`
	Enabled                  *bool                               `json:"enabled"`
	MonitorPath              string                              `json:"monitor_path"`
	RemoteRootPath           string                              `json:"remote_root_path"`
	RemoteRootID             string                              `json:"remote_root_id"`
	Recursive                *bool                               `json:"recursive"`
	WatchMode                models.DirectoryUploadWatchMode     `json:"watch_mode"`
	StartupScanEnabled       *bool                               `json:"startup_scan_enabled"`
	ProcessedCacheTTLSeconds int                                 `json:"processed_cache_ttl_seconds"`
	DeleteSourceAfterSuccess *bool                               `json:"delete_source_after_success"`
	IgnorePatterns           []string                            `json:"ignore_patterns"`
	OverwriteMode            models.DirectoryUploadOverwriteMode `json:"overwrite_mode"`
}

type directoryUploadStatusRequest struct {
	Enabled *bool `json:"enabled"`
}

// ListDirectoryUploadRules 获取目录监控上传规则列表。
func ListDirectoryUploadRules(c *gin.Context) {
	syncPathID, err := parseOptionalUintQuery(c, "sync_path_id")
	if err != nil {
		c.JSON(http.StatusBadRequest, APIResponse[any]{Code: BadRequest, Message: "请求参数错误", Data: nil})
		return
	}
	rules, err := models.GetDirectoryUploadRules(syncPathID)
	if err != nil {
		c.JSON(http.StatusOK, APIResponse[any]{Code: BadRequest, Message: "获取目录监控上传规则失败", Data: nil})
		return
	}
	c.JSON(http.StatusOK, APIResponse[any]{Code: Success, Message: "获取目录监控上传规则成功", Data: gin.H{
		"list":  rules,
		"total": len(rules),
	}})
}

// CreateDirectoryUploadRule 创建目录监控上传规则。
func CreateDirectoryUploadRule(c *gin.Context) {
	var req directoryUploadRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, APIResponse[any]{Code: BadRequest, Message: fmt.Sprintf("请求参数错误：%v", err), Data: nil})
		return
	}
	rule, err := buildDirectoryUploadRuleFromRequest(nil, req)
	if err != nil {
		c.JSON(http.StatusBadRequest, APIResponse[any]{Code: BadRequest, Message: err.Error(), Data: nil})
		return
	}
	if err := createDirectoryUploadRule(rule); err != nil {
		c.JSON(http.StatusOK, APIResponse[any]{Code: BadRequest, Message: "创建目录监控上传规则失败", Data: nil})
		return
	}
	directoryupload.ReloadDirectoryUploadService()
	c.JSON(http.StatusOK, APIResponse[*models.DirectoryUploadRule]{Code: Success, Message: "创建目录监控上传规则成功", Data: rule})
}

// UpdateDirectoryUploadRule 更新目录监控上传规则。
func UpdateDirectoryUploadRule(c *gin.Context) {
	id, ok := parseUintParam(c, "id")
	if !ok {
		return
	}
	var existing models.DirectoryUploadRule
	if err := db.Db.First(&existing, id).Error; err != nil {
		c.JSON(http.StatusNotFound, APIResponse[any]{Code: BadRequest, Message: "目录监控上传规则不存在", Data: nil})
		return
	}
	var req directoryUploadRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, APIResponse[any]{Code: BadRequest, Message: fmt.Sprintf("请求参数错误：%v", err), Data: nil})
		return
	}
	rule, err := buildDirectoryUploadRuleFromRequest(&existing, req)
	if err != nil {
		c.JSON(http.StatusBadRequest, APIResponse[any]{Code: BadRequest, Message: err.Error(), Data: nil})
		return
	}
	if err := db.Db.Save(rule).Error; err != nil {
		c.JSON(http.StatusOK, APIResponse[any]{Code: BadRequest, Message: "更新目录监控上传规则失败", Data: nil})
		return
	}
	directoryupload.ReloadDirectoryUploadService()
	c.JSON(http.StatusOK, APIResponse[*models.DirectoryUploadRule]{Code: Success, Message: "更新目录监控上传规则成功", Data: rule})
}

// DeleteDirectoryUploadRule 删除目录监控上传规则。
func DeleteDirectoryUploadRule(c *gin.Context) {
	id, ok := parseUintParam(c, "id")
	if !ok {
		return
	}
	if err := models.DeleteDirectoryUploadRule(id); err != nil {
		c.JSON(http.StatusOK, APIResponse[any]{Code: BadRequest, Message: "删除目录监控上传规则失败", Data: nil})
		return
	}
	directoryupload.ReloadDirectoryUploadService()
	c.JSON(http.StatusOK, APIResponse[any]{Code: Success, Message: "删除目录监控上传规则成功", Data: nil})
}

// SetDirectoryUploadRuleStatus 设置目录监控上传规则启用状态。
func SetDirectoryUploadRuleStatus(c *gin.Context) {
	id, ok := parseUintParam(c, "id")
	if !ok {
		return
	}
	var req directoryUploadStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.Enabled == nil {
		c.JSON(http.StatusBadRequest, APIResponse[any]{Code: BadRequest, Message: "请求参数错误", Data: nil})
		return
	}
	if _, err := models.GetDirectoryUploadRuleById(id); err != nil {
		c.JSON(http.StatusNotFound, APIResponse[any]{Code: BadRequest, Message: "目录监控上传规则不存在", Data: nil})
		return
	}
	if err := models.SetDirectoryUploadRuleEnabled(id, *req.Enabled); err != nil {
		c.JSON(http.StatusOK, APIResponse[any]{Code: BadRequest, Message: "更新目录监控上传规则状态失败", Data: nil})
		return
	}
	directoryupload.ReloadDirectoryUploadService()
	c.JSON(http.StatusOK, APIResponse[any]{Code: Success, Message: "更新目录监控上传规则状态成功", Data: gin.H{"enabled": *req.Enabled}})
}

// ScanDirectoryUploadRule 手动触发目录监控扫描。
func ScanDirectoryUploadRule(c *gin.Context) {
	id, ok := parseUintParam(c, "id")
	if !ok {
		return
	}
	rule, err := models.GetDirectoryUploadRuleById(id)
	if err != nil {
		c.JSON(http.StatusNotFound, APIResponse[any]{Code: BadRequest, Message: "目录监控上传规则不存在", Data: nil})
		return
	}
	if !rule.Enabled {
		c.JSON(http.StatusOK, APIResponse[any]{Code: BadRequest, Message: "目录监控上传规则未启用", Data: nil})
		return
	}
	accepted, err := directoryupload.ScanRuleNow(context.Background(), rule)
	if err != nil {
		c.JSON(http.StatusOK, APIResponse[any]{Code: BadRequest, Message: err.Error(), Data: nil})
		return
	}
	c.JSON(http.StatusOK, APIResponse[any]{Code: Success, Message: "目录监控扫描已完成", Data: gin.H{"accepted": accepted}})
}

func buildDirectoryUploadRuleFromRequest(existing *models.DirectoryUploadRule, req directoryUploadRuleRequest) (*models.DirectoryUploadRule, error) {
	rule := &models.DirectoryUploadRule{}
	if existing != nil {
		*rule = *existing
	}
	applyDirectoryUploadRuleDefaults(rule, existing == nil)
	if req.SyncPathID > 0 {
		rule.SyncPathId = req.SyncPathID
	}
	if rule.SyncPathId == 0 {
		return nil, fmt.Errorf("同步目录不能为空")
	}
	syncPath := models.GetSyncPathById(rule.SyncPathId)
	if syncPath == nil {
		return nil, fmt.Errorf("同步目录不存在")
	}
	if req.AccountID > 0 {
		rule.AccountId = req.AccountID
	} else if rule.AccountId == 0 {
		rule.AccountId = syncPath.AccountId
	}
	if req.Enabled != nil {
		rule.Enabled = *req.Enabled
	}
	if strings.TrimSpace(req.MonitorPath) != "" {
		rule.MonitorPath = strings.TrimSpace(req.MonitorPath)
	}
	if strings.TrimSpace(req.RemoteRootPath) != "" {
		rule.RemoteRootPath = strings.TrimSpace(req.RemoteRootPath)
	}
	if strings.TrimSpace(req.RemoteRootID) != "" {
		rule.RemoteRootId = strings.TrimSpace(req.RemoteRootID)
	}
	if req.Recursive != nil {
		rule.Recursive = *req.Recursive
	}
	if req.WatchMode != "" {
		rule.WatchMode = req.WatchMode
	}
	if req.StartupScanEnabled != nil {
		rule.StartupScanEnabled = *req.StartupScanEnabled
	}
	if req.ProcessedCacheTTLSeconds > 0 {
		rule.ProcessedCacheTTLSeconds = req.ProcessedCacheTTLSeconds
	}
	if req.DeleteSourceAfterSuccess != nil {
		rule.DeleteSourceAfterSuccess = *req.DeleteSourceAfterSuccess
	}
	if req.OverwriteMode != "" {
		rule.OverwriteMode = req.OverwriteMode
	}
	if req.IgnorePatterns != nil {
		raw, err := json.Marshal(req.IgnorePatterns)
		if err != nil {
			return nil, fmt.Errorf("忽略规则编码失败：%w", err)
		}
		rule.IgnorePatternsStr = string(raw)
	}
	if err := validateDirectoryUploadRuleEnums(rule); err != nil {
		return nil, err
	}
	if err := rule.ValidateWithSyncPath(syncPath); err != nil {
		return nil, err
	}
	return rule, nil
}

func applyDirectoryUploadRuleDefaults(rule *models.DirectoryUploadRule, isCreate bool) {
	if rule.WatchMode == "" {
		rule.WatchMode = models.DirectoryUploadWatchModeAuto
	}
	rule.StabilitySeconds = models.DirectoryUploadDefaultStabilitySeconds
	rule.StabilityCheckIntervalSeconds = models.DirectoryUploadDefaultStabilityCheckIntervalSeconds
	rule.StabilityRequiredCount = models.DirectoryUploadDefaultStabilityRequiredCount
	rule.RescanIntervalSeconds = models.DirectoryUploadDefaultRescanIntervalSeconds
	if rule.ProcessedCacheTTLSeconds <= 0 {
		rule.ProcessedCacheTTLSeconds = 600
	}
	if rule.OverwriteMode == "" {
		rule.OverwriteMode = models.DirectoryUploadOverwriteSkipSame
	}
	if isCreate {
		rule.Enabled = true
		rule.Recursive = true
		rule.StartupScanEnabled = true
	}
}

func validateDirectoryUploadRuleEnums(rule *models.DirectoryUploadRule) error {
	switch rule.WatchMode {
	case models.DirectoryUploadWatchModeAuto, models.DirectoryUploadWatchModeFSNotify, models.DirectoryUploadWatchModePolling:
	default:
		return fmt.Errorf("不支持的监控模式：%s", rule.WatchMode)
	}
	switch rule.OverwriteMode {
	case "",
		models.DirectoryUploadOverwriteSkipSame,
		models.DirectoryUploadOverwriteFailConflict,
		models.DirectoryUploadOverwriteReplaceConflict:
	default:
		return fmt.Errorf("不支持的同名文件处理方式：%s", rule.OverwriteMode)
	}
	return nil
}

func createDirectoryUploadRule(rule *models.DirectoryUploadRule) error {
	enabled := rule.Enabled
	recursive := rule.Recursive
	startupScanEnabled := rule.StartupScanEnabled
	deleteSourceAfterSuccess := rule.DeleteSourceAfterSuccess
	if err := db.Db.Select("*").Create(rule).Error; err != nil {
		return err
	}
	rule.Enabled = enabled
	rule.Recursive = recursive
	rule.StartupScanEnabled = startupScanEnabled
	rule.DeleteSourceAfterSuccess = deleteSourceAfterSuccess
	return db.Db.Model(rule).Updates(map[string]any{
		"enabled":                     enabled,
		"recursive":                   recursive,
		"startup_scan_enabled":        startupScanEnabled,
		"delete_source_after_success": deleteSourceAfterSuccess,
	}).Error
}

func parseOptionalUintQuery(c *gin.Context, name string) (uint, error) {
	raw := strings.TrimSpace(c.Query(name))
	if raw == "" {
		return 0, nil
	}
	value, err := strconv.ParseUint(raw, 10, 64)
	return uint(value), err
}

func parseUintParam(c *gin.Context, name string) (uint, bool) {
	raw := strings.TrimSpace(c.Param(name))
	value, err := strconv.ParseUint(raw, 10, 64)
	if err != nil || value == 0 {
		c.JSON(http.StatusBadRequest, APIResponse[any]{Code: BadRequest, Message: "请求参数错误", Data: nil})
		return 0, false
	}
	return uint(value), true
}
