package models

import (
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"time"

	"qmediasync/internal/db"
	"qmediasync/internal/helpers"

	"gorm.io/gorm"
)

// StrmGenerationSource 是 STRM 生成任务来源。
type StrmGenerationSource string

const (
	StrmGenerationSourceUploadCompleted StrmGenerationSource = "upload_completed"
	StrmGenerationSourceWebhook         StrmGenerationSource = "webhook"
	StrmGenerationSourceRemoteExists    StrmGenerationSource = "remote_exists"
)

// StrmGenerationTaskType 是 STRM 生成任务类型。
type StrmGenerationTaskType string

const (
	StrmGenerationTaskTypeFile          StrmGenerationTaskType = "file"
	StrmGenerationTaskTypeDirectoryScan StrmGenerationTaskType = "directory_scan"
	StrmGenerationTaskTypeBatchFiles    StrmGenerationTaskType = "batch_files"
)

// StrmGenerationStatus 是 STRM 生成任务状态。
type StrmGenerationStatus string

const (
	StrmGenerationStatusPending         StrmGenerationStatus = "pending"
	StrmGenerationStatusRunning         StrmGenerationStatus = "running"
	StrmGenerationStatusWaitingChildren StrmGenerationStatus = "waiting_children"
	StrmGenerationStatusCompleted       StrmGenerationStatus = "completed"
	StrmGenerationStatusFailed          StrmGenerationStatus = "failed"
	StrmGenerationStatusCancelled       StrmGenerationStatus = "cancelled"
)

// StrmGenerationTask 保存上传完成或 Webhook 触发的 STRM 生成任务。
type StrmGenerationTask struct {
	BaseModel
	Source       StrmGenerationSource   `json:"source" gorm:"size:32;index"`
	TaskType     StrmGenerationTaskType `json:"task_type" gorm:"size:32;index"`
	ParentTaskId uint                   `json:"parent_task_id" gorm:"index"`
	UploadTaskId uint                   `json:"upload_task_id" gorm:"index"`
	SyncPathId   uint                   `json:"sync_path_id" gorm:"index"`
	AccountId    uint                   `json:"account_id" gorm:"index"`

	DownloadMeta bool `json:"download_meta" gorm:"default:false"`
	RefreshEmby  bool `json:"refresh_emby" gorm:"default:false"`

	FileId   string `json:"file_id" gorm:"size:128;index"`
	ParentId string `json:"parent_id" gorm:"size:128"`
	PickCode string `json:"pick_code" gorm:"size:128;index"`
	Path     string `json:"path" gorm:"type:text;size:1024"`
	FileName string `json:"file_name" gorm:"size:512"`
	FileSize int64  `json:"file_size"`
	Sha1     string `json:"sha1" gorm:"size:64"`
	Mtime    int64  `json:"mtime"`

	DirectoryId   string `json:"directory_id" gorm:"size:128;index"`
	DirectoryPath string `json:"directory_path" gorm:"type:text;size:1024"`
	TotalItems    int    `json:"total_items"`
	AcceptedItems int    `json:"accepted_items"`
	FailedItems   int    `json:"failed_items"`
	ChangedItems  int    `json:"changed_items"`
	NewMetaItems  int    `json:"new_meta_items"`

	RefreshTargetsStr string `json:"-" gorm:"type:text;default:'[]'"`
	RefreshSubmitted  bool   `json:"refresh_submitted" gorm:"default:false"`

	Status        StrmGenerationStatus `json:"status" gorm:"size:32;index"`
	RequestHash   string               `json:"request_hash" gorm:"size:255;uniqueIndex:idx_strm_generation_request_hash,where:request_hash <> ''"`
	RetryCount    int                  `json:"retry_count" gorm:"default:0"`
	LastRetryTime int64                `json:"last_retry_time"`
	LastError     string               `json:"last_error" gorm:"type:text"`
}

// EnqueueStrmGenerationTask 创建 STRM 生成任务，request_hash 非空时做幂等去重。
func EnqueueStrmGenerationTask(task *StrmGenerationTask) (*StrmGenerationTask, error) {
	return EnqueueStrmGenerationTaskWithDB(db.Db, task)
}

// EnqueueStrmGenerationTaskWithDB 在指定事务中创建 STRM 生成任务，request_hash 非空时做幂等去重。
func EnqueueStrmGenerationTaskWithDB(tx *gorm.DB, task *StrmGenerationTask) (*StrmGenerationTask, error) {
	if tx == nil {
		return nil, errors.New("数据库连接为空")
	}
	if task == nil {
		return nil, errors.New("STRM 生成任务为空")
	}
	if task.RequestHash != "" {
		var existing StrmGenerationTask
		err := tx.Where("request_hash = ?", task.RequestHash).First(&existing).Error
		if err == nil {
			if !isActiveStrmGenerationStatus(existing.Status) {
				archivedHash := archivedStrmGenerationRequestHash(task.RequestHash, existing.ID)
				if err := tx.Model(&StrmGenerationTask{}).
					Where("id = ? AND request_hash = ?", existing.ID, task.RequestHash).
					Update("request_hash", archivedHash).Error; err != nil {
					return nil, err
				}
			} else {
				return &existing, nil
			}
		} else {
			if !errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, err
			}
		}
	}
	if task.Status == "" {
		task.Status = StrmGenerationStatusPending
	}
	if task.TaskType == "" {
		task.TaskType = StrmGenerationTaskTypeFile
	}
	if err := tx.Create(task).Error; err != nil {
		return nil, err
	}
	return task, nil
}

// StrmGenerationParentProgress 描述父任务需要累计的子任务结果。
type StrmGenerationParentProgress struct {
	Accepted       int
	Failed         int
	Changed        int
	NewMeta        int
	RefreshTargets []EmbyRefreshTarget
}

// GetRefreshTargets 返回父任务已累计的 Emby 刷新目标。
func (task *StrmGenerationTask) GetRefreshTargets() []EmbyRefreshTarget {
	var targets []EmbyRefreshTarget
	if task == nil || task.RefreshTargetsStr == "" {
		return targets
	}
	if err := json.Unmarshal([]byte(task.RefreshTargetsStr), &targets); err != nil {
		helpers.AppLogger.Warnf("解析 STRM 生成任务刷新目标失败：%v", err)
		return []EmbyRefreshTarget{}
	}
	return normalizeEmbyRefreshTargets(targets)
}

// SetRefreshTargets 保存父任务已累计的 Emby 刷新目标。
func (task *StrmGenerationTask) SetRefreshTargets(targets []EmbyRefreshTarget) {
	data, err := json.Marshal(normalizeEmbyRefreshTargets(targets))
	if err != nil {
		task.RefreshTargetsStr = "[]"
		return
	}
	task.RefreshTargetsStr = string(data)
}

// MergeRefreshTargets 合并并去重父任务 Emby 刷新目标。
func (task *StrmGenerationTask) MergeRefreshTargets(targets []EmbyRefreshTarget) {
	if task == nil || len(targets) == 0 {
		return
	}
	task.SetRefreshTargets(append(task.GetRefreshTargets(), targets...))
}

// IsReadyToSubmitRefresh 判断父任务是否已经可以提交一次批量 Emby 刷新。
func (task *StrmGenerationTask) IsReadyToSubmitRefresh() bool {
	if task == nil || !task.RefreshEmby || task.RefreshSubmitted {
		return false
	}
	if task.ChangedItems == 0 && task.NewMetaItems == 0 {
		return false
	}
	if task.TotalItems == 0 {
		return false
	}
	return task.AcceptedItems+task.FailedItems >= task.TotalItems
}

func normalizeEmbyRefreshTargets(targets []EmbyRefreshTarget) []EmbyRefreshTarget {
	libraries := make(map[string]EmbyRefreshTarget)
	for _, target := range targets {
		if target.TargetType != EmbyRefreshTargetTypeLibrary {
			continue
		}
		key := refreshTargetLibraryKey(target)
		if existing, ok := libraries[key]; ok {
			libraries[key] = mergeRefreshLibraryTarget(existing, target)
			continue
		}
		libraries[key] = target
	}

	items := make(map[string]EmbyRefreshTarget)
	for _, target := range targets {
		if target.TargetType == "" {
			target.TargetType = EmbyRefreshTargetTypeLibrary
		}
		if target.TargetType == EmbyRefreshTargetTypeLibrary {
			continue
		}
		if target.TargetType != EmbyRefreshTargetTypeItem || target.ItemID == "" {
			continue
		}
		if _, covered := libraries[refreshTargetLibraryKey(target)]; covered {
			continue
		}
		key := string(target.TargetType) + ":" + target.ItemID
		if existing, ok := items[key]; ok {
			items[key] = mergeRefreshItemTarget(existing, target)
			continue
		}
		items[key] = target
	}

	result := make([]EmbyRefreshTarget, 0, len(libraries)+len(items))
	for _, target := range libraries {
		result = append(result, target)
	}
	for _, target := range items {
		result = append(result, target)
	}
	sort.Slice(result, func(i, j int) bool {
		return refreshTargetSortKey(result[i]) < refreshTargetSortKey(result[j])
	})
	return result
}

func refreshTargetLibraryKey(target EmbyRefreshTarget) string {
	if target.FallbackLibraryId != "" {
		return target.FallbackLibraryId
	}
	if target.SyncPathID > 0 {
		return fmt.Sprintf("sync_path:%d", target.SyncPathID)
	}
	return "__default_library__"
}

func refreshTargetSortKey(target EmbyRefreshTarget) string {
	switch target.TargetType {
	case EmbyRefreshTargetTypeLibrary:
		return "0:" + refreshTargetLibraryKey(target)
	default:
		return "1:" + target.FallbackLibraryId + ":" + target.ItemID
	}
}

func mergeRefreshLibraryTarget(left EmbyRefreshTarget, right EmbyRefreshTarget) EmbyRefreshTarget {
	if left.FallbackLibraryId == "" {
		left.FallbackLibraryId = right.FallbackLibraryId
	}
	if left.FallbackLibraryName == "" {
		left.FallbackLibraryName = right.FallbackLibraryName
	}
	if left.SyncPathID == 0 {
		left.SyncPathID = right.SyncPathID
	}
	return left
}

func mergeRefreshItemTarget(left EmbyRefreshTarget, right EmbyRefreshTarget) EmbyRefreshTarget {
	if left.ItemName == "" {
		left.ItemName = right.ItemName
	}
	if left.ItemType == "" {
		left.ItemType = right.ItemType
	}
	left.Recursive = left.Recursive || right.Recursive
	if left.SyncPathID == 0 {
		left.SyncPathID = right.SyncPathID
	}
	if left.FallbackLibraryId == "" {
		left.FallbackLibraryId = right.FallbackLibraryId
	}
	if left.FallbackLibraryName == "" {
		left.FallbackLibraryName = right.FallbackLibraryName
	}
	return left
}

func isActiveStrmGenerationStatus(status StrmGenerationStatus) bool {
	return status == StrmGenerationStatusPending ||
		status == StrmGenerationStatusRunning ||
		status == StrmGenerationStatusWaitingChildren
}

func archivedStrmGenerationRequestHash(requestHash string, taskID uint) string {
	const maxRequestHashLength = 255
	suffix := fmt.Sprintf(":history:%d", taskID)
	if len(requestHash)+len(suffix) <= maxRequestHashLength {
		return requestHash + suffix
	}
	keep := maxRequestHashLength - len(suffix)
	if keep <= 0 {
		return suffix[:maxRequestHashLength]
	}
	return requestHash[:keep] + suffix
}

// MarkFailed 标记 STRM 生成任务失败并累计重试次数。
func (task *StrmGenerationTask) MarkFailed(message string) error {
	if task == nil {
		return errors.New("STRM 生成任务为空")
	}
	task.Status = StrmGenerationStatusFailed
	task.RetryCount++
	task.LastRetryTime = time.Now().Unix()
	task.LastError = message
	return db.Db.Save(task).Error
}

// MarkRunning 标记 STRM 生成任务开始执行。
func (task *StrmGenerationTask) MarkRunning() error {
	if task == nil {
		return errors.New("STRM 生成任务为空")
	}
	result := db.Db.Model(task).
		Where("id = ? AND status = ?", task.ID, StrmGenerationStatusPending).
		Updates(map[string]interface{}{
			"status":     StrmGenerationStatusRunning,
			"last_error": "",
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errors.New("STRM 生成任务状态不可执行")
	}
	task.Status = StrmGenerationStatusRunning
	task.LastError = ""
	return nil
}

// MarkCompleted 标记 STRM 生成任务执行完成。
func (task *StrmGenerationTask) MarkCompleted() error {
	if task == nil {
		return errors.New("STRM 生成任务为空")
	}
	task.Status = StrmGenerationStatusCompleted
	task.LastError = ""
	return db.Db.Save(task).Error
}

// GetPendingStrmGenerationTasks 按创建顺序获取待执行 STRM 生成任务。
func GetPendingStrmGenerationTasks(limit int) ([]*StrmGenerationTask, error) {
	if limit <= 0 {
		limit = 10
	}
	var tasks []*StrmGenerationTask
	err := db.Db.Where("status = ?", StrmGenerationStatusPending).
		Order("id ASC").
		Limit(limit).
		Find(&tasks).Error
	return tasks, err
}

// ResetRunningStrmGenerationTasks 将进程退出前的运行中任务恢复为待处理。
func ResetRunningStrmGenerationTasks() error {
	return db.Db.Model(&StrmGenerationTask{}).
		Where("status = ?", StrmGenerationStatusRunning).
		Update("status", StrmGenerationStatusPending).Error
}

// IncrementStrmGenerationDirectoryStats 累加目录扫描任务统计。
func IncrementStrmGenerationDirectoryStats(parentTaskId uint, accepted int, failed int) error {
	_, err := UpdateStrmGenerationParentProgress(parentTaskId, StrmGenerationParentProgress{
		Accepted: accepted,
		Failed:   failed,
	})
	return err
}

// UpdateStrmGenerationParentProgress 以事务方式累计父任务进度和刷新目标。
func UpdateStrmGenerationParentProgress(parentTaskId uint, progress StrmGenerationParentProgress) (*StrmGenerationTask, error) {
	if parentTaskId == 0 {
		return nil, nil
	}
	var updated StrmGenerationTask
	err := db.Db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&StrmGenerationTask{}).
			Where("id = ?", parentTaskId).
			Updates(map[string]interface{}{
				"accepted_items": gorm.Expr("accepted_items + ?", progress.Accepted),
				"failed_items":   gorm.Expr("failed_items + ?", progress.Failed),
				"changed_items":  gorm.Expr("changed_items + ?", progress.Changed),
				"new_meta_items": gorm.Expr("new_meta_items + ?", progress.NewMeta),
			}).Error; err != nil {
			return err
		}

		var parent StrmGenerationTask
		if err := tx.First(&parent, parentTaskId).Error; err != nil {
			return err
		}
		processedItems := parent.AcceptedItems + parent.FailedItems
		hasFixedTotalItems := parent.TotalItems > 0
		updates := map[string]interface{}{}
		if parent.TotalItems < processedItems {
			parent.TotalItems = processedItems
			updates["total_items"] = parent.TotalItems
		}
		if hasFixedTotalItems &&
			processedItems >= parent.TotalItems &&
			parent.Status == StrmGenerationStatusWaitingChildren {
			if parent.FailedItems > 0 {
				parent.Status = StrmGenerationStatusFailed
				updates["status"] = parent.Status
			} else {
				parent.Status = StrmGenerationStatusCompleted
				parent.LastError = ""
				updates["status"] = parent.Status
				updates["last_error"] = parent.LastError
			}
		}
		if len(progress.RefreshTargets) > 0 {
			parent.MergeRefreshTargets(progress.RefreshTargets)
			updates["refresh_targets_str"] = parent.RefreshTargetsStr
		}
		if len(updates) > 0 {
			if err := tx.Model(&StrmGenerationTask{}).
				Where("id = ?", parentTaskId).
				Updates(updates).Error; err != nil {
				return err
			}
			if err := tx.First(&parent, parentTaskId).Error; err != nil {
				return err
			}
		}
		updated = parent
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &updated, nil
}

// MarkStrmGenerationRefreshSubmitted 标记父任务刷新目标已经提交。
func MarkStrmGenerationRefreshSubmitted(parentTaskId uint) error {
	if parentTaskId == 0 {
		return nil
	}
	return db.Db.Model(&StrmGenerationTask{}).
		Where("id = ?", parentTaskId).
		Update("refresh_submitted", true).Error
}
