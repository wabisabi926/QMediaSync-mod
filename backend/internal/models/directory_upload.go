package models

import (
	"encoding/json"
	"errors"
	"fmt"
	pathpkg "path"
	"path/filepath"
	"strings"

	"qmediasync/internal/db"

	"gorm.io/gorm"
)

// DirectoryUploadWatchMode 是目录监控模式。
type DirectoryUploadWatchMode string

const (
	DirectoryUploadWatchModeAuto     DirectoryUploadWatchMode = "auto"
	DirectoryUploadWatchModeFSNotify DirectoryUploadWatchMode = "fsnotify"
	DirectoryUploadWatchModePolling  DirectoryUploadWatchMode = "polling"
)

// DirectoryUploadOverwriteMode 是远端已有文件处理策略。
type DirectoryUploadOverwriteMode string

const (
	DirectoryUploadOverwriteSkipSame        DirectoryUploadOverwriteMode = "skip_same"
	DirectoryUploadOverwriteFailConflict    DirectoryUploadOverwriteMode = "fail_conflict"
	DirectoryUploadOverwriteReplaceConflict DirectoryUploadOverwriteMode = "replace_conflict"
)

// 目录监控上传内置计时参数。
const (
	DirectoryUploadDefaultStabilitySeconds              = 15
	DirectoryUploadDefaultStabilityCheckIntervalSeconds = 2
	DirectoryUploadDefaultStabilityRequiredCount        = 3
	DirectoryUploadDefaultRescanIntervalSeconds         = 30
)

// DirectoryUploadRule 保存目录监控上传规则。
type DirectoryUploadRule struct {
	BaseModel
	SyncPathId                    uint                         `json:"sync_path_id" gorm:"index"`
	AccountId                     uint                         `json:"account_id" gorm:"index"`
	Enabled                       bool                         `json:"enabled" gorm:"default:true;index"`
	MonitorPath                   string                       `json:"monitor_path" gorm:"type:text;size:1024"`
	RemoteRootPath                string                       `json:"remote_root_path" gorm:"type:text;size:1024"`
	RemoteRootId                  string                       `json:"remote_root_id" gorm:"size:128"`
	Recursive                     bool                         `json:"recursive" gorm:"default:true"`
	UploadMetadata                bool                         `json:"upload_metadata" gorm:"default:false"`
	WatchMode                     DirectoryUploadWatchMode     `json:"watch_mode" gorm:"size:32;default:auto"`
	StabilitySeconds              int                          `json:"stability_seconds" gorm:"default:15"`
	StabilityCheckIntervalSeconds int                          `json:"stability_check_interval_seconds" gorm:"default:2"`
	StabilityRequiredCount        int                          `json:"stability_required_count" gorm:"default:3"`
	RescanIntervalSeconds         int                          `json:"rescan_interval_seconds" gorm:"default:30"`
	StartupScanEnabled            bool                         `json:"startup_scan_enabled" gorm:"default:true"`
	ProcessedCacheTTLSeconds      int                          `json:"processed_cache_ttl_seconds" gorm:"default:600"`
	DeleteSourceAfterSuccess      bool                         `json:"delete_source_after_success" gorm:"default:false"`
	IgnorePatterns                []string                     `json:"ignore_patterns" gorm:"-"`
	IgnorePatternsStr             string                       `json:"-" gorm:"type:text"`
	OverwriteMode                 DirectoryUploadOverwriteMode `json:"overwrite_mode" gorm:"size:32;default:skip_same"`
}

// SaveDirectoryUploadRule 保存目录监控上传规则。
func SaveDirectoryUploadRule(rule *DirectoryUploadRule) error {
	if rule == nil {
		return errors.New("目录监控上传规则为空")
	}
	if rule.IgnorePatterns != nil {
		if err := rule.SetIgnorePatterns(rule.IgnorePatterns); err != nil {
			return err
		}
	}
	rule.applyDefaults()

	syncPath := GetSyncPathById(rule.SyncPathId)
	if syncPath == nil {
		return fmt.Errorf("同步目录 %d 不存在", rule.SyncPathId)
	}
	if err := rule.ValidateWithSyncPath(syncPath); err != nil {
		return err
	}
	if err := db.Db.Save(rule).Error; err != nil {
		return err
	}
	rule.LoadIgnorePatterns()
	return nil
}

// GetEnabledDirectoryUploadRules 查询启用的目录监控上传规则。
func GetEnabledDirectoryUploadRules() ([]*DirectoryUploadRule, error) {
	var rules []*DirectoryUploadRule
	if err := db.Db.Where("enabled = ?", true).Order("id ASC").Find(&rules).Error; err != nil {
		return nil, err
	}
	loadDirectoryUploadRuleIgnorePatterns(rules)
	return rules, nil
}

// GetDirectoryUploadRuleById 按 ID 查询目录监控上传规则。
func GetDirectoryUploadRuleById(id uint) (*DirectoryUploadRule, error) {
	var rule DirectoryUploadRule
	if err := db.Db.First(&rule, id).Error; err != nil {
		return nil, err
	}
	rule.LoadIgnorePatterns()
	return &rule, nil
}

// GetDirectoryUploadRules 查询目录监控上传规则列表。
func GetDirectoryUploadRules(syncPathID uint) ([]*DirectoryUploadRule, error) {
	var rules []*DirectoryUploadRule
	query := db.Db.Order("id ASC")
	if syncPathID > 0 {
		query = query.Where("sync_path_id = ?", syncPathID)
	}
	if err := query.Find(&rules).Error; err != nil {
		return nil, err
	}
	loadDirectoryUploadRuleIgnorePatterns(rules)
	return rules, nil
}

// DeleteDirectoryUploadRule 删除目录监控上传规则。
func DeleteDirectoryUploadRule(id uint) error {
	return db.Db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Delete(&DirectoryUploadRule{}, id).Error; err != nil {
			return err
		}
		return tx.Where("rule_id = ?", id).Delete(&DirectoryUploadProcessedFile{}).Error
	})
}

// SetDirectoryUploadRuleEnabled 设置目录监控上传规则启用状态。
func SetDirectoryUploadRuleEnabled(id uint, enabled bool) error {
	return db.Db.Model(&DirectoryUploadRule{}).Where("id = ?", id).Update("enabled", enabled).Error
}

// ValidateWithSyncPath 校验目录监控上传规则和同步目录的路径边界。
func (rule *DirectoryUploadRule) ValidateWithSyncPath(syncPath *SyncPath) error {
	if rule == nil {
		return errors.New("目录监控上传规则为空")
	}
	if syncPath == nil {
		return errors.New("同步目录为空")
	}

	monitorPath := cleanLocalPath(rule.MonitorPath)
	localPath := cleanLocalPath(syncPath.LocalPath)
	if monitorPath == "" {
		return errors.New("监控目录不能为空")
	}
	if localPath != "" && monitorPath == localPath {
		return errors.New("监控目录不能等于 STRM 本地目录")
	}

	remoteRootPath := cleanRemotePath(rule.RemoteRootPath)
	syncRemotePath := cleanRemotePath(syncPath.RemotePath)
	if remoteRootPath == "" {
		return errors.New("远端上传根目录不能为空")
	}
	if syncRemotePath == "" {
		return errors.New("同步远端目录不能为空")
	}
	if !isRemotePathWithin(remoteRootPath, syncRemotePath) {
		return fmt.Errorf("远端上传根目录 %s 不在同步远端目录 %s 下", remoteRootPath, syncRemotePath)
	}
	return nil
}

// SetIgnorePatterns 设置并编码目录监控上传忽略规则。
func (rule *DirectoryUploadRule) SetIgnorePatterns(patterns []string) error {
	if rule == nil {
		return errors.New("目录监控上传规则为空")
	}
	normalized := normalizeDirectoryUploadIgnorePatterns(patterns)
	raw, err := json.Marshal(normalized)
	if err != nil {
		return fmt.Errorf("忽略规则编码失败：%w", err)
	}
	rule.IgnorePatterns = normalized
	rule.IgnorePatternsStr = string(raw)
	return nil
}

// LoadIgnorePatterns 解析目录监控上传忽略规则。
func (rule *DirectoryUploadRule) LoadIgnorePatterns() {
	if rule == nil {
		return
	}
	rule.IgnorePatterns = parseDirectoryUploadIgnorePatterns(rule.IgnorePatternsStr)
}

func (rule *DirectoryUploadRule) applyDefaults() {
	if rule.WatchMode == "" {
		rule.WatchMode = DirectoryUploadWatchModeAuto
	}
	rule.StabilitySeconds = DirectoryUploadDefaultStabilitySeconds
	rule.StabilityCheckIntervalSeconds = DirectoryUploadDefaultStabilityCheckIntervalSeconds
	rule.StabilityRequiredCount = DirectoryUploadDefaultStabilityRequiredCount
	rule.RescanIntervalSeconds = DirectoryUploadDefaultRescanIntervalSeconds
	if rule.ProcessedCacheTTLSeconds <= 0 {
		rule.ProcessedCacheTTLSeconds = 600
	}
	if rule.OverwriteMode == "" {
		rule.OverwriteMode = DirectoryUploadOverwriteSkipSame
	}
	if rule.ID == 0 {
		rule.Recursive = true
		rule.StartupScanEnabled = true
	}
}

func cleanLocalPath(p string) string {
	p = strings.TrimSpace(p)
	if p == "" {
		return ""
	}
	return filepath.Clean(p)
}

func cleanRemotePath(p string) string {
	p = strings.TrimSpace(strings.ReplaceAll(p, "\\", "/"))
	if p == "" {
		return ""
	}
	if !strings.HasPrefix(p, "/") {
		p = "/" + p
	}
	return pathpkg.Clean(p)
}

func isRemotePathWithin(remotePath string, basePath string) bool {
	if basePath == "/" {
		return strings.HasPrefix(remotePath, "/")
	}
	return remotePath == basePath || strings.HasPrefix(remotePath, basePath+"/")
}

func loadDirectoryUploadRuleIgnorePatterns(rules []*DirectoryUploadRule) {
	for _, rule := range rules {
		rule.LoadIgnorePatterns()
	}
}

func parseDirectoryUploadIgnorePatterns(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return []string{}
	}
	var patterns []string
	if err := json.Unmarshal([]byte(raw), &patterns); err == nil {
		return normalizeDirectoryUploadIgnorePatterns(patterns)
	}
	fields := strings.FieldsFunc(raw, func(r rune) bool {
		return r == '\n' || r == '\r' || r == ',' || r == ';'
	})
	return normalizeDirectoryUploadIgnorePatterns(fields)
}

func normalizeDirectoryUploadIgnorePatterns(patterns []string) []string {
	result := make([]string, 0, len(patterns))
	for _, pattern := range patterns {
		pattern = strings.TrimSpace(pattern)
		if pattern != "" {
			result = append(result, pattern)
		}
	}
	return result
}
