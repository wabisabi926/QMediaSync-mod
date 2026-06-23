package models

import (
	"Q115-STRM/internal/db"
	"Q115-STRM/internal/helpers"
	"encoding/json"
	"errors"
	"sort"
	"strings"
	"time"
	"gorm.io/gorm"
)

const (
	EmbyLibraryRefreshStatusPending    = "pending"
	EmbyLibraryRefreshStatusRefreshing = "refreshing"
	EmbyLibraryRefreshStatusCompleted  = "completed"
	EmbyLibraryRefreshStatusFailed     = "failed"

	DefaultEmbyRefreshDebounceSeconds = int64(10)
	DefaultEmbyRefreshMaxWaitSeconds  = int64(60 * 60)
	DefaultEmbyRefreshScanSeconds     = int64(60)
)

var IsStrmSyncTaskActiveFunc func(syncPathId uint) bool

type EmbyLibraryRefreshTask struct {
	BaseModel
	LibraryId      string `json:"library_id" gorm:"uniqueIndex;type:varchar(128)"`
	LibraryName    string `json:"library_name" gorm:"type:varchar(255)"`
	SyncPathIdsStr string `json:"-" gorm:"type:text;default:'[]'"`
	Status         string `json:"status" gorm:"index;type:varchar(32)"`
	LastEventAt    int64  `json:"last_event_at" gorm:"index"`
	RefreshAfterAt int64  `json:"refresh_after_at" gorm:"index"`
	DeadlineAt     int64  `json:"deadline_at" gorm:"index"`
	LastCheckedAt  int64  `json:"last_checked_at"`
	LastRefreshAt  int64  `json:"last_refresh_at"`
	Error          string `json:"error" gorm:"type:text"`
}

func (*EmbyLibraryRefreshTask) TableName() string {
	return "emby_library_refresh_tasks"
}

func (t *EmbyLibraryRefreshTask) GetSyncPathIds() []uint {
	var ids []uint
	if t == nil || t.SyncPathIdsStr == "" {
		return ids
	}
	if err := json.Unmarshal([]byte(t.SyncPathIdsStr), &ids); err != nil {
		helpers.AppLogger.Warnf("解析Emby媒体库刷新任务sync_path_ids失败: %v", err)
		return []uint{}
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	return ids
}

func (t *EmbyLibraryRefreshTask) SetSyncPathIds(ids []uint) {
	merged := mergeSyncPathIds(ids, nil)
	data, err := json.Marshal(merged)
	if err != nil {
		t.SyncPathIdsStr = "[]"
		return
	}
	t.SyncPathIdsStr = string(data)
}

func mergeSyncPathIds(left []uint, right []uint) []uint {
	seen := make(map[uint]bool)
	for _, id := range left {
		if id > 0 {
			seen[id] = true
		}
	}
	for _, id := range right {
		if id > 0 {
			seen[id] = true
		}
	}
	ids := make([]uint, 0, len(seen))
	for id := range seen {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	return ids
}

func newPendingEmbyLibraryRefreshTask(libraryId string, libraryName string, syncPathIds []uint, now int64) *EmbyLibraryRefreshTask {
	task := &EmbyLibraryRefreshTask{
		LibraryId:      libraryId,
		LibraryName:    libraryName,
		Status:         EmbyLibraryRefreshStatusPending,
		LastEventAt:    now,
		RefreshAfterAt: now + DefaultEmbyRefreshDebounceSeconds,
		DeadlineAt:     now + DefaultEmbyRefreshMaxWaitSeconds,
	}
	task.SetSyncPathIds(syncPathIds)
	return task
}

func saveEmbyLibraryRefreshTask(task *EmbyLibraryRefreshTask) error {
	return db.Db.Save(task).Error
}

func RequestEmbyLibraryRefreshBySyncPathId(syncPathId uint) error {
	if syncPathId == 0 {
		helpers.AppLogger.Infof("临时同步路径不触发Emby媒体库刷新")
		return nil
	}
	if GlobalEmbyConfig == nil || GlobalEmbyConfig.EmbyUrl == "" || GlobalEmbyConfig.EmbyApiKey == "" || GlobalEmbyConfig.EnableRefreshLibrary == 0 {
		helpers.AppLogger.Infof("Emby未配置或未启用刷新媒体库，跳过提交刷新任务")
		return nil
	}

	libraries := GetEmbyLibraryIdsBySyncPathId(syncPathId)
	if len(libraries) == 0 {
		helpers.AppLogger.Infof("同步目录 %d 未关联Emby媒体库，跳过提交刷新任务", syncPathId)
		return nil
	}

	now := nowUnix()
	for libraryId, libraryName := range libraries {
		if err := upsertEmbyLibraryRefreshTask(libraryId, libraryName, syncPathId, now); err != nil {
			return err
		}
	}
	TriggerEmbyLibraryRefreshCheck()
	return nil
}

func upsertEmbyLibraryRefreshTask(libraryId string, libraryName string, syncPathId uint, now int64) error {
	return db.Db.Transaction(func(tx *gorm.DB) error {
		return upsertEmbyLibraryRefreshTaskWithDB(tx, libraryId, libraryName, syncPathId, now)
	})
}

func upsertEmbyLibraryRefreshTaskWithDB(tx *gorm.DB, libraryId string, libraryName string, syncPathId uint, now int64) error {
	var task EmbyLibraryRefreshTask
	err := tx.Where("library_id = ?", libraryId).First(&task).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		task = *newPendingEmbyLibraryRefreshTask(libraryId, libraryName, []uint{syncPathId}, now)
		if err := tx.Create(&task).Error; err != nil {
			if isUniqueConstraintError(err) {
				return upsertEmbyLibraryRefreshTaskWithDB(tx, libraryId, libraryName, syncPathId, now)
			}
			return err
		}
		return nil
	}
	if err != nil {
		return err
	}

	existingIds := task.GetSyncPathIds()
	mergedIds := mergeSyncPathIds(existingIds, []uint{syncPathId})
	oldStatus := task.Status
	task.LibraryName = libraryName
	task.Status = EmbyLibraryRefreshStatusPending
	task.LastEventAt = now
	task.RefreshAfterAt = now + DefaultEmbyRefreshDebounceSeconds
	if len(mergedIds) > len(existingIds) || task.DeadlineAt <= now || oldStatus == EmbyLibraryRefreshStatusCompleted || oldStatus == EmbyLibraryRefreshStatusFailed {
		task.DeadlineAt = now + DefaultEmbyRefreshMaxWaitSeconds
	}
	task.LastCheckedAt = 0
	task.Error = ""
	task.SetSyncPathIds(mergedIds)
	return tx.Save(&task).Error
}

func isUniqueConstraintError(err error) bool {
	if err == nil {
		return false
	}
	message := err.Error()
	return errors.Is(err, gorm.ErrDuplicatedKey) ||
		strings.Contains(message, "UNIQUE constraint failed: emby_library_refresh_tasks.library_id")
}

func TriggerEmbyLibraryRefreshCheck() {
}

func nowUnix() int64 {
	return time.Now().Unix()
}
