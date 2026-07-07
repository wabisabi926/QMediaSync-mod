package directoryupload

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"qmediasync/internal/db"
	"qmediasync/internal/helpers"
	"qmediasync/internal/models"

	"gorm.io/gorm"
)

// CleanupSourceAfterStrmSuccess 在目录监控上传和 STRM 生成都成功后清理本地源文件。
func CleanupSourceAfterStrmSuccess(uploadTaskID uint) error {
	if uploadTaskID == 0 {
		return nil
	}
	var task models.DbUploadTask
	if err := db.Db.First(&task, uploadTaskID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		return err
	}
	if task.Source != models.UploadSourceDirectoryMonitor {
		return nil
	}
	if task.SourceCleanupStatus != models.UploadSourceCleanupStatusPending {
		return nil
	}
	rule, err := findCleanupRule(&task)
	if err != nil {
		return markCleanupFailed(&task, err)
	}
	if rule == nil || !rule.DeleteSourceAfterSuccess {
		return markCleanupNone(&task)
	}
	if !isSafeCleanupUploadTask(&task) {
		return markCleanupNone(&task)
	}
	hasCompletedStrm, err := hasCompletedStrmTask(task.ID)
	if err != nil {
		return err
	}
	if !hasCompletedStrm {
		return nil
	}
	if err := ensurePathWithinRoot(rule.MonitorPath, task.LocalFullPath); err != nil {
		return markCleanupFailed(&task, err)
	}
	if err := validateCurrentSourceFileForCleanup(&task); err != nil {
		return markCleanupFailed(&task, err)
	}
	if err := os.Remove(task.LocalFullPath); err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return markCleanupFailed(&task, fmt.Errorf("删除源文件失败：%w", err))
		}
	} else if helpers.AppLogger != nil {
		helpers.AppLogger.Infof(
			"[目录上传] 已删除源文件：upload_task_id=%d rule_id=%d path=%s result=%s remote_file_id=%s",
			task.ID,
			rule.ID,
			task.LocalFullPath,
			task.UploadResult,
			task.CompletedRemoteFileId,
		)
	}
	if err := removeEmptyParents(filepath.Dir(task.LocalFullPath), rule.MonitorPath, task.ID, rule.ID); err != nil {
		return markCleanupFailed(&task, err)
	}
	task.SourceCleanupStatus = models.UploadSourceCleanupStatusCompleted
	task.SourceCleanupError = ""
	task.SourceDeletedAt = time.Now().Unix()
	return db.Db.Save(&task).Error
}

func findCleanupRule(task *models.DbUploadTask) (*models.DirectoryUploadRule, error) {
	if task == nil || task.SyncPathId == 0 || task.LocalFullPath == "" {
		return nil, nil
	}
	rule, found, err := findCleanupRuleByProcessedTask(task.ID)
	if err != nil || found {
		return rule, err
	}
	var rules []*models.DirectoryUploadRule
	if err := db.Db.Where("sync_path_id = ?", task.SyncPathId).Find(&rules).Error; err != nil {
		return nil, err
	}
	var selected *models.DirectoryUploadRule
	selectedLen := -1
	for _, rule := range rules {
		if rule == nil {
			continue
		}
		if err := ensurePathWithinRoot(rule.MonitorPath, task.LocalFullPath); err != nil {
			continue
		}
		cleanLen := len(filepath.Clean(rule.MonitorPath))
		if cleanLen > selectedLen {
			selected = rule
			selectedLen = cleanLen
		}
	}
	if selected == nil && len(rules) > 0 {
		return nil, fmt.Errorf("源文件路径不在目录监控规则内：%s", task.LocalFullPath)
	}
	return selected, nil
}

func findCleanupRuleByProcessedTask(uploadTaskID uint) (*models.DirectoryUploadRule, bool, error) {
	if uploadTaskID == 0 {
		return nil, false, nil
	}
	var record models.DirectoryUploadProcessedFile
	if err := db.Db.Where("upload_task_id = ?", uploadTaskID).First(&record).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, false, nil
		}
		return nil, true, err
	}
	if record.RuleId == 0 {
		return nil, true, fmt.Errorf("目录监控处理记录缺少规则 ID：upload_task_id=%d", uploadTaskID)
	}
	rule, err := models.GetDirectoryUploadRuleById(record.RuleId)
	if err != nil {
		return nil, true, fmt.Errorf("读取目录监控清理规则失败：%w", err)
	}
	return rule, true, nil
}

func isSafeCleanupUploadTask(task *models.DbUploadTask) bool {
	if task == nil || task.Status != models.UploadStatusCompleted {
		return false
	}
	switch task.UploadResult {
	case models.UploadResultRapidUpload, models.UploadResultMultipartUploaded:
		return task.CompletedRemoteFileId != "" || task.CompletedPickCode != ""
	case models.UploadResultRemoteExists:
		return task.CompletedRemoteFileId != "" || task.CompletedPickCode != ""
	default:
		return false
	}
}

func validateCurrentSourceFileForCleanup(task *models.DbUploadTask) error {
	if task == nil {
		return errors.New("上传任务为空")
	}
	info, err := os.Stat(task.LocalFullPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("读取源文件信息失败：%w", err)
	}
	expectedFingerprint := strings.TrimSpace(task.SourceFingerprint)
	if expectedFingerprint == "" {
		return errors.New("上传任务缺少源文件 fingerprint，跳过删除")
	}
	currentFingerprint := models.BuildDirectoryUploadSourceFingerprint(info.Size(), info.ModTime().UnixNano())
	if currentFingerprint != expectedFingerprint {
		return fmt.Errorf("源文件已变化，跳过删除：fingerprint 不匹配 task=%s current=%s", expectedFingerprint, currentFingerprint)
	}
	return nil
}

func hasCompletedStrmTask(uploadTaskID uint) (bool, error) {
	var total int64
	err := db.Db.Model(&models.StrmGenerationTask{}).
		Where("upload_task_id = ? AND status = ?", uploadTaskID, models.StrmGenerationStatusCompleted).
		Count(&total).Error
	return total > 0, err
}

func ensurePathWithinRoot(rootPath string, targetPath string) error {
	rootPath = filepath.Clean(rootPath)
	targetPath = filepath.Clean(targetPath)
	rel, err := filepath.Rel(rootPath, targetPath)
	if err != nil {
		return fmt.Errorf("计算路径边界失败：%w", err)
	}
	if rel == "." || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || filepath.IsAbs(rel) {
		return fmt.Errorf("源文件路径越界：%s", targetPath)
	}
	return nil
}

func removeEmptyParents(startDir string, rootPath string, uploadTaskID uint, ruleID uint) error {
	rootPath = filepath.Clean(rootPath)
	dir := filepath.Clean(startDir)
	for {
		if dir == rootPath {
			return nil
		}
		if err := ensurePathWithinRoot(rootPath, dir); err != nil {
			return err
		}
		entries, err := os.ReadDir(dir)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				dir = filepath.Dir(dir)
				continue
			}
			return fmt.Errorf("读取待清理目录失败：%w", err)
		}
		if len(entries) > 0 {
			return nil
		}
		if err := os.Remove(dir); err != nil {
			if errors.Is(err, os.ErrNotExist) {
				dir = filepath.Dir(dir)
				continue
			}
			return fmt.Errorf("删除空目录失败：%w", err)
		}
		if helpers.AppLogger != nil {
			helpers.AppLogger.Infof(
				"[目录上传] 已删除空目录：upload_task_id=%d rule_id=%d path=%s",
				uploadTaskID,
				ruleID,
				dir,
			)
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return nil
		}
		dir = parent
	}
}

func markCleanupNone(task *models.DbUploadTask) error {
	if task == nil {
		return nil
	}
	if task.SourceCleanupStatus == models.UploadSourceCleanupStatusNone && task.SourceCleanupError == "" {
		return nil
	}
	task.SourceCleanupStatus = models.UploadSourceCleanupStatusNone
	task.SourceCleanupError = ""
	return db.Db.Save(task).Error
}

func markCleanupFailed(task *models.DbUploadTask, err error) error {
	if task == nil || err == nil {
		return err
	}
	task.SourceCleanupStatus = models.UploadSourceCleanupStatusFailed
	task.SourceCleanupError = err.Error()
	if saveErr := db.Db.Save(task).Error; saveErr != nil {
		return saveErr
	}
	return err
}
