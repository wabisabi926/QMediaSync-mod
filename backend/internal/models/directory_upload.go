package models

import (
	"errors"
	"fmt"
	pathpkg "path"
	"path/filepath"
	"strings"

	"qmediasync/internal/db"
)

// DirectoryUploadWatchMode 是目录监控模式。
type DirectoryUploadWatchMode string

const (
	DirectoryUploadWatchModeAuto    DirectoryUploadWatchMode = "auto"
	DirectoryUploadWatchModeWatcher DirectoryUploadWatchMode = "watcher"
	DirectoryUploadWatchModePolling DirectoryUploadWatchMode = "polling"
)

// DirectoryUploadOverwriteMode 是远端已有文件处理策略。
type DirectoryUploadOverwriteMode string

const (
	DirectoryUploadOverwriteSkipSame DirectoryUploadOverwriteMode = "skip_same"
	DirectoryUploadOverwriteAlways   DirectoryUploadOverwriteMode = "always"
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
	WatchMode                     DirectoryUploadWatchMode     `json:"watch_mode" gorm:"size:32;default:auto"`
	StabilitySeconds              int                          `json:"stability_seconds" gorm:"default:15"`
	StabilityCheckIntervalSeconds int                          `json:"stability_check_interval_seconds" gorm:"default:2"`
	StabilityRequiredCount        int                          `json:"stability_required_count" gorm:"default:3"`
	RescanIntervalSeconds         int                          `json:"rescan_interval_seconds" gorm:"default:300"`
	StartupScanEnabled            bool                         `json:"startup_scan_enabled" gorm:"default:true"`
	ProcessedCacheTTLSeconds      int                          `json:"processed_cache_ttl_seconds" gorm:"default:600"`
	DeleteSourceAfterSuccess      bool                         `json:"delete_source_after_success" gorm:"default:false"`
	IgnorePatternsStr             string                       `json:"-" gorm:"type:text"`
	OverwriteMode                 DirectoryUploadOverwriteMode `json:"overwrite_mode" gorm:"size:32;default:skip_same"`
}

// SaveDirectoryUploadRule 保存目录监控上传规则。
func SaveDirectoryUploadRule(rule *DirectoryUploadRule) error {
	if rule == nil {
		return errors.New("目录监控上传规则为空")
	}
	rule.applyDefaults()

	syncPath := GetSyncPathById(rule.SyncPathId)
	if syncPath == nil {
		return fmt.Errorf("同步目录 %d 不存在", rule.SyncPathId)
	}
	if err := rule.ValidateWithSyncPath(syncPath); err != nil {
		return err
	}
	return db.Db.Save(rule).Error
}

// GetEnabledDirectoryUploadRules 查询启用的目录监控上传规则。
func GetEnabledDirectoryUploadRules() ([]*DirectoryUploadRule, error) {
	var rules []*DirectoryUploadRule
	if err := db.Db.Where("enabled = ?", true).Order("id ASC").Find(&rules).Error; err != nil {
		return nil, err
	}
	return rules, nil
}

// GetDirectoryUploadRuleById 按 ID 查询目录监控上传规则。
func GetDirectoryUploadRuleById(id uint) (*DirectoryUploadRule, error) {
	var rule DirectoryUploadRule
	if err := db.Db.First(&rule, id).Error; err != nil {
		return nil, err
	}
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
	return rules, nil
}

// DeleteDirectoryUploadRule 删除目录监控上传规则。
func DeleteDirectoryUploadRule(id uint) error {
	return db.Db.Delete(&DirectoryUploadRule{}, id).Error
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

func (rule *DirectoryUploadRule) applyDefaults() {
	if rule.WatchMode == "" {
		rule.WatchMode = DirectoryUploadWatchModeAuto
	}
	if rule.StabilitySeconds <= 0 {
		rule.StabilitySeconds = 15
	}
	if rule.StabilityCheckIntervalSeconds <= 0 {
		rule.StabilityCheckIntervalSeconds = 2
	}
	if rule.StabilityRequiredCount <= 0 {
		rule.StabilityRequiredCount = 3
	}
	if rule.RescanIntervalSeconds <= 0 {
		rule.RescanIntervalSeconds = 300
	}
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
