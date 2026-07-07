package models

import (
	"encoding/json"
	"errors"
	"fmt"
	pathpkg "path"
	"path/filepath"
	"runtime"
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
	if err := ValidateDirectoryUploadRuleScope(rule); err != nil {
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
	if err := db.Db.
		Model(&DirectoryUploadRule{}).
		Joins("JOIN sync_paths ON sync_paths.id = directory_upload_rules.sync_path_id").
		Where("directory_upload_rules.enabled = ? AND sync_paths.directory_upload_enabled = ?", true, true).
		Order("directory_upload_rules.id ASC").
		Find(&rules).Error; err != nil {
		return nil, err
	}
	loadDirectoryUploadRuleIgnorePatterns(rules)
	return rules, nil
}

// GetEnabledDirectoryUploadRulesBySyncPathId 查询指定同步目录下启用的目录监控上传规则。
func GetEnabledDirectoryUploadRulesBySyncPathId(syncPathID uint) ([]*DirectoryUploadRule, error) {
	var rules []*DirectoryUploadRule
	if err := db.Db.
		Model(&DirectoryUploadRule{}).
		Joins("JOIN sync_paths ON sync_paths.id = directory_upload_rules.sync_path_id").
		Where("directory_upload_rules.sync_path_id = ? AND directory_upload_rules.enabled = ?", syncPathID, true).
		Where("sync_paths.directory_upload_enabled = ?", true).
		Order("directory_upload_rules.id ASC").
		Find(&rules).Error; err != nil {
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

// SaveDirectoryUploadRulesForSyncPath 批量保存同步目录下目录监控上传规则。
func SaveDirectoryUploadRulesForSyncPath(syncPathID uint, enabled bool, rules []*DirectoryUploadRule) ([]*DirectoryUploadRule, error) {
	syncPath := GetSyncPathById(syncPathID)
	if syncPath == nil {
		return nil, fmt.Errorf("同步目录不存在")
	}

	var saved []*DirectoryUploadRule
	err := db.Db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&SyncPath{}).
			Where("id = ?", syncPathID).
			Update("directory_upload_enabled", enabled).Error; err != nil {
			return err
		}
		syncPath.DirectoryUploadEnabled = enabled

		existingRules, err := getDirectoryUploadRulesWithDB(tx, syncPathID)
		if err != nil {
			return err
		}
		existingByID := make(map[uint]*DirectoryUploadRule, len(existingRules))
		for _, rule := range existingRules {
			existingByID[rule.ID] = rule
		}

		finalRules := make([]*DirectoryUploadRule, 0, len(rules))
		seenRuleIDs := make(map[uint]struct{}, len(rules))
		for _, rule := range rules {
			if rule == nil {
				return errors.New("目录监控上传规则为空")
			}
			if rule.SyncPathId != syncPathID {
				return fmt.Errorf("目录监控上传规则 %d 不属于当前同步目录", rule.ID)
			}
			if rule.ID > 0 {
				if _, seen := seenRuleIDs[rule.ID]; seen {
					return fmt.Errorf("目录监控上传规则 %d 重复提交", rule.ID)
				}
				seenRuleIDs[rule.ID] = struct{}{}
				if _, ok := existingByID[rule.ID]; !ok {
					return fmt.Errorf("目录监控上传规则 %d 不属于当前同步目录", rule.ID)
				}
			}
			if rule.IgnorePatterns != nil {
				if err := rule.SetIgnorePatterns(rule.IgnorePatterns); err != nil {
					return err
				}
			}
			if err := rule.ValidateWithSyncPath(syncPath); err != nil {
				return err
			}
			finalRules = append(finalRules, rule)
		}
		if enabled && !hasEnabledDirectoryUploadRule(finalRules) {
			return errors.New("目录监控上传已启用，请至少启用一条规则")
		}
		if err := validateEnabledDirectoryUploadRuleSet(finalRules); err != nil {
			return err
		}

		submittedIDSet := make(map[uint]struct{}, len(seenRuleIDs))
		for id := range seenRuleIDs {
			submittedIDSet[id] = struct{}{}
		}
		deletedIDs := make([]uint, 0)
		for _, rule := range existingRules {
			if _, ok := submittedIDSet[rule.ID]; !ok {
				deletedIDs = append(deletedIDs, rule.ID)
			}
		}
		if len(deletedIDs) > 0 {
			if err := tx.Where("id IN ?", deletedIDs).Delete(&DirectoryUploadRule{}).Error; err != nil {
				return err
			}
			if err := tx.Where("rule_id IN ?", deletedIDs).Delete(&DirectoryUploadProcessedFile{}).Error; err != nil {
				return err
			}
		}
		for _, rule := range rules {
			if rule.ID == 0 {
				if err := createDirectoryUploadRuleWithDB(tx, rule); err != nil {
					return err
				}
				continue
			}
			if err := tx.Save(rule).Error; err != nil {
				return err
			}
		}
		saved, err = getDirectoryUploadRulesWithDB(tx, syncPathID)
		return err
	})
	if err != nil {
		return nil, err
	}
	return saved, nil
}

func getDirectoryUploadRulesWithDB(handle *gorm.DB, syncPathID uint) ([]*DirectoryUploadRule, error) {
	var rules []*DirectoryUploadRule
	if err := handle.
		Where("sync_path_id = ?", syncPathID).
		Order("id ASC").
		Find(&rules).Error; err != nil {
		return nil, err
	}
	loadDirectoryUploadRuleIgnorePatterns(rules)
	return rules, nil
}

func createDirectoryUploadRuleWithDB(handle *gorm.DB, rule *DirectoryUploadRule) error {
	enabled := rule.Enabled
	recursive := rule.Recursive
	uploadMetadata := rule.UploadMetadata
	startupScanEnabled := rule.StartupScanEnabled
	deleteSourceAfterSuccess := rule.DeleteSourceAfterSuccess
	if err := handle.Select("*").Create(rule).Error; err != nil {
		return err
	}
	rule.Enabled = enabled
	rule.Recursive = recursive
	rule.UploadMetadata = uploadMetadata
	rule.StartupScanEnabled = startupScanEnabled
	rule.DeleteSourceAfterSuccess = deleteSourceAfterSuccess
	return handle.Model(rule).Updates(map[string]any{
		"enabled":                     enabled,
		"recursive":                   recursive,
		"upload_metadata":             uploadMetadata,
		"startup_scan_enabled":        startupScanEnabled,
		"delete_source_after_success": deleteSourceAfterSuccess,
	}).Error
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

// ValidateDirectoryUploadRuleScope 校验同一同步目录下目录监控规则是否重复或重叠。
func ValidateDirectoryUploadRuleScope(rule *DirectoryUploadRule) error {
	if rule == nil {
		return errors.New("目录监控上传规则为空")
	}
	var rules []*DirectoryUploadRule
	query := db.Db.Where("sync_path_id = ?", rule.SyncPathId)
	if rule.ID > 0 {
		query = query.Where("id <> ?", rule.ID)
	}
	if err := query.Order("id ASC").Find(&rules).Error; err != nil {
		return fmt.Errorf("查询目录监控上传规则失败：%w", err)
	}
	for _, existing := range rules {
		if err := validateDirectoryUploadRulePair(rule, existing); err != nil {
			return err
		}
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

func cleanLocalPathForDirectoryUploadCompare(p string) string {
	p = strings.TrimSpace(p)
	if p == "" {
		return ""
	}
	p = strings.ReplaceAll(p, "\\", string(filepath.Separator))
	p = filepath.Clean(p)
	if runtime.GOOS == "windows" {
		p = strings.ToLower(p)
	}
	return p
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

func validateEnabledDirectoryUploadRuleSet(rules []*DirectoryUploadRule) error {
	for i, rule := range rules {
		for _, existing := range rules[i+1:] {
			if err := validateDirectoryUploadRulePair(rule, existing); err != nil {
				return err
			}
		}
	}
	return nil
}

func hasEnabledDirectoryUploadRule(rules []*DirectoryUploadRule) bool {
	for _, rule := range rules {
		if rule != nil && rule.Enabled {
			return true
		}
	}
	return false
}

func validateDirectoryUploadRulePair(rule *DirectoryUploadRule, existing *DirectoryUploadRule) error {
	if rule == nil || existing == nil {
		return nil
	}
	monitorPath := cleanLocalPathForDirectoryUploadCompare(rule.MonitorPath)
	existingMonitorPath := cleanLocalPathForDirectoryUploadCompare(existing.MonitorPath)
	remoteRootPath := cleanRemotePath(rule.RemoteRootPath)
	existingRemoteRootPath := cleanRemotePath(existing.RemoteRootPath)
	remoteRootID := strings.TrimSpace(rule.RemoteRootId)
	existingRemoteRootID := strings.TrimSpace(existing.RemoteRootId)

	if monitorPath == existingMonitorPath &&
		remoteRootPath == existingRemoteRootPath &&
		remoteRootID == existingRemoteRootID {
		return fmt.Errorf("目录监控上传规则重复：监控目录 %s 已绑定到目标目录 %s", rule.MonitorPath, rule.RemoteRootPath)
	}
	if !rule.Enabled || !existing.Enabled {
		return nil
	}
	if monitorPath == existingMonitorPath {
		return fmt.Errorf("目录监控上传规则重叠：监控目录 %s 已被规则 %d 使用", rule.MonitorPath, existing.ID)
	}
	if existing.Recursive && isLocalPathWithinDirectoryUploadMonitor(monitorPath, existingMonitorPath) {
		return fmt.Errorf("目录监控上传规则重叠：监控目录 %s 位于递归监控目录 %s 下", rule.MonitorPath, existing.MonitorPath)
	}
	if rule.Recursive && isLocalPathWithinDirectoryUploadMonitor(existingMonitorPath, monitorPath) {
		return fmt.Errorf("目录监控上传规则重叠：已有监控目录 %s 位于递归监控目录 %s 下", existing.MonitorPath, rule.MonitorPath)
	}
	return nil
}

func isLocalPathWithinDirectoryUploadMonitor(path string, monitorPath string) bool {
	if path == "" || monitorPath == "" || path == monitorPath {
		return false
	}
	rel, err := filepath.Rel(monitorPath, path)
	if err != nil {
		return false
	}
	return rel != "." &&
		rel != ".." &&
		!strings.HasPrefix(rel, ".."+string(filepath.Separator)) &&
		!filepath.IsAbs(rel)
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
