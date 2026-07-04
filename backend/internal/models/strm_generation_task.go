package models

import (
	"errors"
	"fmt"
	"time"

	"qmediasync/internal/db"

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
)

// StrmGenerationStatus 是 STRM 生成任务状态。
type StrmGenerationStatus string

const (
	StrmGenerationStatusPending   StrmGenerationStatus = "pending"
	StrmGenerationStatusRunning   StrmGenerationStatus = "running"
	StrmGenerationStatusCompleted StrmGenerationStatus = "completed"
	StrmGenerationStatusFailed    StrmGenerationStatus = "failed"
	StrmGenerationStatusCancelled StrmGenerationStatus = "cancelled"
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

	Status        StrmGenerationStatus `json:"status" gorm:"size:32;index"`
	RequestHash   string               `json:"request_hash" gorm:"size:255;uniqueIndex:idx_strm_generation_request_hash,where:request_hash <> ''"`
	RetryCount    int                  `json:"retry_count" gorm:"default:0"`
	LastRetryTime int64                `json:"last_retry_time"`
	LastError     string               `json:"last_error" gorm:"type:text"`
}

// EnqueueStrmGenerationTask 创建 STRM 生成任务，request_hash 非空时做幂等去重。
func EnqueueStrmGenerationTask(task *StrmGenerationTask) (*StrmGenerationTask, error) {
	if task == nil {
		return nil, errors.New("STRM 生成任务为空")
	}
	if task.RequestHash != "" {
		var existing StrmGenerationTask
		err := db.Db.Where("request_hash = ?", task.RequestHash).First(&existing).Error
		if err == nil {
			if !isActiveStrmGenerationStatus(existing.Status) {
				archivedHash := archivedStrmGenerationRequestHash(task.RequestHash, existing.ID)
				if err := db.Db.Model(&StrmGenerationTask{}).
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
	if err := db.Db.Create(task).Error; err != nil {
		return nil, err
	}
	return task, nil
}

func isActiveStrmGenerationStatus(status StrmGenerationStatus) bool {
	return status == StrmGenerationStatusPending || status == StrmGenerationStatusRunning
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
	var parent StrmGenerationTask
	if err := db.Db.First(&parent, parentTaskId).Error; err != nil {
		return err
	}
	parent.AcceptedItems += accepted
	parent.FailedItems += failed
	processedItems := parent.AcceptedItems + parent.FailedItems
	if parent.TotalItems < processedItems {
		parent.TotalItems = processedItems
	}
	return db.Db.Save(&parent).Error
}
