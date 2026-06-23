package models

import (
	"Q115-STRM/internal/db"
	"Q115-STRM/internal/helpers"
	"encoding/json"
	"sort"
	"time"
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

func nowUnix() int64 {
	return time.Now().Unix()
}
