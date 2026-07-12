package controllers

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"qmediasync/internal/directoryupload"
	"qmediasync/internal/models"

	"github.com/gin-gonic/gin"
)

type directoryUploadRuleRequest struct {
	ClientID                 string                              `json:"client_id"`
	ID                       uint                                `json:"id"`
	SyncPathID               uint                                `json:"sync_path_id"`
	AccountID                uint                                `json:"account_id"`
	Enabled                  *bool                               `json:"enabled"`
	MonitorPath              string                              `json:"monitor_path"`
	RemoteRootPath           string                              `json:"remote_root_path"`
	RemoteRootID             string                              `json:"remote_root_id"`
	Recursive                *bool                               `json:"recursive"`
	UploadMetadata           *bool                               `json:"upload_metadata"`
	WatchMode                models.DirectoryUploadWatchMode     `json:"watch_mode"`
	StartupScanEnabled       *bool                               `json:"startup_scan_enabled"`
	ProcessedCacheTTLSeconds int                                 `json:"processed_cache_ttl_seconds"`
	DeleteSourceAfterSuccess *bool                               `json:"delete_source_after_success"`
	IgnorePatterns           []string                            `json:"ignore_patterns"`
	OverwriteMode            models.DirectoryUploadOverwriteMode `json:"overwrite_mode"`
}

type directoryUploadRulesSaveRequest struct {
	Enabled *bool                         `json:"enabled"`
	Rules   *[]directoryUploadRuleRequest `json:"rules"`
}

type directoryUploadRuleScanItem struct {
	RuleID   uint   `json:"rule_id"`
	Accepted int    `json:"accepted"`
	Error    string `json:"error,omitempty"`
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

// ScanDirectoryUploadSyncPathRules 手动触发同步目录下所有启用目录监控规则扫描。
func ScanDirectoryUploadSyncPathRules(c *gin.Context) {
	syncPathID, ok := parseUintParam(c, "sync_path_id")
	if !ok {
		return
	}
	if models.GetSyncPathById(syncPathID) == nil {
		c.JSON(http.StatusNotFound, APIResponse[any]{Code: BadRequest, Message: "同步目录不存在", Data: nil})
		return
	}
	rules, err := models.GetEnabledDirectoryUploadRulesBySyncPathId(syncPathID)
	if err != nil {
		c.JSON(http.StatusOK, APIResponse[any]{Code: BadRequest, Message: "获取目录监控上传规则失败", Data: nil})
		return
	}
	if len(rules) == 0 {
		c.JSON(http.StatusOK, APIResponse[any]{Code: BadRequest, Message: "目录监控上传未启用", Data: nil})
		return
	}

	scanCtx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Minute)
	defer cancel()
	items := make([]directoryUploadRuleScanItem, 0, len(rules))
	totalAccepted := 0
	var firstErr error
	for _, rule := range rules {
		accepted, scanErr := directoryupload.ScanRuleNow(scanCtx, rule)
		item := directoryUploadRuleScanItem{RuleID: rule.ID, Accepted: accepted}
		if scanErr != nil {
			item.Error = scanErr.Error()
			if firstErr == nil {
				firstErr = scanErr
			}
		}
		totalAccepted += accepted
		items = append(items, item)
	}
	data := gin.H{"accepted": totalAccepted, "items": items}
	if firstErr != nil {
		c.JSON(http.StatusOK, APIResponse[any]{Code: BadRequest, Message: firstErr.Error(), Data: data})
		return
	}
	c.JSON(http.StatusOK, APIResponse[any]{Code: Success, Message: "目录监控扫描已完成", Data: data})
}

// GetDirectoryUploadRuntimeStatuses 获取目录监控运行状态。
func GetDirectoryUploadRuntimeStatuses(c *gin.Context) {
	statuses := directoryupload.GetDirectoryUploadRuntimeStatuses()
	c.JSON(http.StatusOK, APIResponse[any]{
		Code:    Success,
		Message: "获取目录监控运行状态成功",
		Data:    gin.H{"items": statuses},
	})
}

func buildDirectoryUploadRuleFromRequestWithoutScopeValidation(existing *models.DirectoryUploadRule, req directoryUploadRuleRequest) (*models.DirectoryUploadRule, error) {
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
	rule.MonitorPath = strings.TrimSpace(req.MonitorPath)
	rule.RemoteRootPath = strings.TrimSpace(req.RemoteRootPath)
	rule.RemoteRootId = strings.TrimSpace(req.RemoteRootID)
	if req.Recursive != nil {
		rule.Recursive = *req.Recursive
	}
	if req.UploadMetadata != nil {
		rule.UploadMetadata = *req.UploadMetadata
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
		if err := rule.SetIgnorePatterns(req.IgnorePatterns); err != nil {
			return nil, err
		}
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
