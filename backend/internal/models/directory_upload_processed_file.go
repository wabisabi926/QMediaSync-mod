package models

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	pathpkg "path"
	"path/filepath"
	"strings"
	"time"

	"qmediasync/internal/db"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// DirectoryUploadProcessedResult 是目录监控源文件处理结果。
type DirectoryUploadProcessedResult string

const (
	DirectoryUploadProcessedResultQueued          DirectoryUploadProcessedResult = "queued"
	DirectoryUploadProcessedResultUploaded        DirectoryUploadProcessedResult = "uploaded"
	DirectoryUploadProcessedResultRemoteExists    DirectoryUploadProcessedResult = "remote_exists"
	DirectoryUploadProcessedResultSkippedExisting DirectoryUploadProcessedResult = "skipped_existing"
	DirectoryUploadProcessedResultFailed          DirectoryUploadProcessedResult = "failed"
)

// DirectoryUploadProcessedFile 保存目录监控源文件处理账本。
type DirectoryUploadProcessedFile struct {
	BaseModel
	RuleId            uint                           `json:"rule_id" gorm:"index"`
	SyncPathId        uint                           `json:"sync_path_id" gorm:"index"`
	AccountId         uint                           `json:"account_id" gorm:"index"`
	ScopeHash         string                         `json:"scope_hash" gorm:"size:64;index"`
	SourceKey         string                         `json:"source_key" gorm:"size:64;uniqueIndex"`
	RelativePath      string                         `json:"relative_path" gorm:"type:text;size:1024"`
	LocalFullPath     string                         `json:"local_full_path" gorm:"type:text;size:1024"`
	SourceFingerprint string                         `json:"source_fingerprint" gorm:"size:128;index"`
	FileSize          int64                          `json:"file_size"`
	LocalMtimeNs      int64                          `json:"local_mtime_ns" gorm:"default:0"`
	Result            DirectoryUploadProcessedResult `json:"result" gorm:"size:32;index"`
	UploadTaskId      uint                           `json:"upload_task_id" gorm:"index"`
	ProcessedAt       int64                          `json:"processed_at" gorm:"index"`
	LastSeenAt        int64                          `json:"last_seen_at" gorm:"index"`
}

// BuildDirectoryUploadSourceFingerprint 生成目录监控源文件签名。
// 当前 v1 只包含 size 和 mtime_ns，不包含 ctime、inode 或文件 hash。
func BuildDirectoryUploadSourceFingerprint(size int64, mtimeNs int64) string {
	return fmt.Sprintf("v1:%d:%d", size, mtimeNs)
}

func directoryUploadHash(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

// BuildDirectoryUploadScopeHash 生成目录监控规则处理范围哈希。
func BuildDirectoryUploadScopeHash(rule *DirectoryUploadRule) string {
	if rule == nil {
		return ""
	}
	monitorPath := filepath.ToSlash(filepath.Clean(rule.MonitorPath))
	remoteRootPath := pathpkg.Clean(strings.ReplaceAll(rule.RemoteRootPath, "\\", "/"))
	remoteRootID := strings.TrimSpace(rule.RemoteRootId)
	raw := fmt.Sprintf(
		"v1\nrule=%d\nsync_path=%d\naccount=%d\nmonitor=%s\nremote_root=%s\nremote_root_id=%s",
		rule.ID,
		rule.SyncPathId,
		rule.AccountId,
		monitorPath,
		remoteRootPath,
		remoteRootID,
	)
	return directoryUploadHash(raw)
}

// BuildDirectoryUploadSourceKey 生成目录监控源文件在规则范围内的稳定键。
func BuildDirectoryUploadSourceKey(scopeHash string, relativePath string) string {
	rel := strings.ReplaceAll(relativePath, "\\", "/")
	rel = filepath.ToSlash(filepath.Clean(rel))
	return directoryUploadHash(scopeHash + "\n" + rel)
}

// IsDirectoryUploadProcessedTerminal 判断处理结果是否为可跳过的终态。
func IsDirectoryUploadProcessedTerminal(result DirectoryUploadProcessedResult) bool {
	switch result {
	case DirectoryUploadProcessedResultUploaded,
		DirectoryUploadProcessedResultRemoteExists,
		DirectoryUploadProcessedResultSkippedExisting:
		return true
	default:
		return false
	}
}

// FindDirectoryUploadProcessedBySourceKey 按源文件稳定键查询处理记录。
func FindDirectoryUploadProcessedBySourceKey(sourceKey string) (*DirectoryUploadProcessedFile, error) {
	if strings.TrimSpace(sourceKey) == "" {
		return nil, errors.New("目录监控源文件 source_key 为空")
	}
	var record DirectoryUploadProcessedFile
	if err := db.Db.Where("source_key = ?", sourceKey).First(&record).Error; err != nil {
		return nil, err
	}
	return &record, nil
}

// UpsertDirectoryUploadProcessedFile 创建或更新目录监控源文件处理记录。
func UpsertDirectoryUploadProcessedFile(record *DirectoryUploadProcessedFile) error {
	return UpsertDirectoryUploadProcessedFileWithDB(db.Db, record)
}

// UpsertDirectoryUploadProcessedFileWithDB 在指定事务中创建或更新目录监控源文件处理记录。
func UpsertDirectoryUploadProcessedFileWithDB(tx *gorm.DB, record *DirectoryUploadProcessedFile) error {
	if tx == nil {
		return errors.New("数据库连接为空")
	}
	if record == nil {
		return errors.New("目录监控源文件处理记录为空")
	}
	if strings.TrimSpace(record.SourceKey) == "" {
		return errors.New("目录监控源文件 source_key 为空")
	}
	return tx.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "source_key"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"rule_id",
			"sync_path_id",
			"account_id",
			"scope_hash",
			"relative_path",
			"local_full_path",
			"source_fingerprint",
			"file_size",
			"local_mtime_ns",
			"result",
			"upload_task_id",
			"processed_at",
			"last_seen_at",
			"updated_at",
		}),
	}).Create(record).Error
}

// MarkDirectoryUploadProcessedUploaded 标记目录监控上传任务关联源文件已上传。
func MarkDirectoryUploadProcessedUploaded(uploadTaskID uint, result DirectoryUploadProcessedResult) error {
	return MarkDirectoryUploadProcessedUploadedWithDB(db.Db, uploadTaskID, result)
}

// MarkDirectoryUploadProcessedUploadedWithDB 在指定事务中标记目录监控上传任务关联源文件已上传。
func MarkDirectoryUploadProcessedUploadedWithDB(tx *gorm.DB, uploadTaskID uint, result DirectoryUploadProcessedResult) error {
	if tx == nil {
		return errors.New("数据库连接为空")
	}
	if uploadTaskID == 0 {
		return errors.New("上传任务 ID 为空")
	}
	now := time.Now().Unix()
	return tx.Model(&DirectoryUploadProcessedFile{}).
		Where("upload_task_id = ?", uploadTaskID).
		Updates(map[string]any{
			"result":       result,
			"processed_at": now,
			"last_seen_at": now,
			"updated_at":   now,
		}).Error
}

// DeleteDirectoryUploadProcessedFilesByRuleID 删除指定规则的源文件处理记录。
func DeleteDirectoryUploadProcessedFilesByRuleID(ruleID uint) error {
	return db.Db.Where("rule_id = ?", ruleID).Delete(&DirectoryUploadProcessedFile{}).Error
}

// CleanupDirectoryUploadProcessedFiles 清理可安全重试或源文件已消失的处理记录。
func CleanupDirectoryUploadProcessedFiles(now time.Time, missingSourceTTL time.Duration) (int64, error) {
	cutoff := now.Add(-missingSourceTTL).Unix()
	var deleted int64

	failedDeleted, err := cleanupDirectoryUploadProcessedFailed(cutoff)
	if err != nil {
		return deleted, err
	}
	deleted += failedDeleted

	queuedDeleted, err := cleanupDirectoryUploadProcessedQueued()
	if err != nil {
		return deleted, err
	}
	deleted += queuedDeleted

	successDeleted, err := cleanupDirectoryUploadProcessedSuccessful(cutoff)
	if err != nil {
		return deleted, err
	}
	deleted += successDeleted

	return deleted, nil
}

func cleanupDirectoryUploadProcessedFailed(cutoff int64) (int64, error) {
	var deleted int64
	for {
		var records []DirectoryUploadProcessedFile
		if err := db.Db.
			Where("result = ? AND processed_at <= ?", DirectoryUploadProcessedResultFailed, cutoff).
			Order("id ASC").
			Limit(500).
			Find(&records).Error; err != nil {
			return deleted, err
		}
		if len(records) == 0 {
			return deleted, nil
		}
		batchDeleted, err := deleteDirectoryUploadProcessedBatch(records)
		if err != nil {
			return deleted, err
		}
		deleted += batchDeleted
		if len(records) < 500 {
			return deleted, nil
		}
	}
}

func cleanupDirectoryUploadProcessedQueued() (int64, error) {
	var deleted int64
	var lastID uint
	for {
		var records []DirectoryUploadProcessedFile
		if err := db.Db.
			Where("result = ? AND id > ?", DirectoryUploadProcessedResultQueued, lastID).
			Order("id ASC").
			Limit(500).
			Find(&records).Error; err != nil {
			return deleted, err
		}
		if len(records) == 0 {
			return deleted, nil
		}
		ids := make([]uint, 0, len(records))
		for _, record := range records {
			lastID = record.ID
			if shouldDeleteQueuedDirectoryUploadProcessed(record) {
				ids = append(ids, record.ID)
			}
		}
		batchDeleted, err := deleteDirectoryUploadProcessedIDs(ids)
		if err != nil {
			return deleted, err
		}
		deleted += batchDeleted
		if len(records) < 500 {
			return deleted, nil
		}
	}
}

func cleanupDirectoryUploadProcessedSuccessful(cutoff int64) (int64, error) {
	var deleted int64
	var lastID uint
	for {
		var records []DirectoryUploadProcessedFile
		if err := db.Db.
			Where("result IN ? AND last_seen_at <= ? AND id > ?", []DirectoryUploadProcessedResult{
				DirectoryUploadProcessedResultUploaded,
				DirectoryUploadProcessedResultRemoteExists,
				DirectoryUploadProcessedResultSkippedExisting,
			}, cutoff, lastID).
			Order("id ASC").
			Limit(500).
			Find(&records).Error; err != nil {
			return deleted, err
		}
		if len(records) == 0 {
			return deleted, nil
		}
		ids := make([]uint, 0, len(records))
		for _, record := range records {
			lastID = record.ID
			if _, err := os.Stat(record.LocalFullPath); err != nil && os.IsNotExist(err) {
				ids = append(ids, record.ID)
			}
		}
		batchDeleted, err := deleteDirectoryUploadProcessedIDs(ids)
		if err != nil {
			return deleted, err
		}
		deleted += batchDeleted
		if len(records) < 500 {
			return deleted, nil
		}
	}
}

func shouldDeleteQueuedDirectoryUploadProcessed(record DirectoryUploadProcessedFile) bool {
	if record.UploadTaskId == 0 {
		return true
	}
	var task DbUploadTask
	if err := db.Db.Select("id", "status").First(&task, record.UploadTaskId).Error; err != nil {
		return errors.Is(err, gorm.ErrRecordNotFound)
	}
	return task.Status != UploadStatusPending && task.Status != UploadStatusUploading
}

func deleteDirectoryUploadProcessedBatch(records []DirectoryUploadProcessedFile) (int64, error) {
	ids := make([]uint, 0, len(records))
	for _, record := range records {
		ids = append(ids, record.ID)
	}
	return deleteDirectoryUploadProcessedIDs(ids)
}

func deleteDirectoryUploadProcessedIDs(ids []uint) (int64, error) {
	if len(ids) == 0 {
		return 0, nil
	}
	result := db.Db.Where("id IN ?", ids).Delete(&DirectoryUploadProcessedFile{})
	return result.RowsAffected, result.Error
}
