package models

import (
	"Q115-STRM/internal/db"
	embyclientrestgo "Q115-STRM/internal/embyclient-rest-go"
	"Q115-STRM/internal/helpers"
	"encoding/json"
	"errors"
	"sort"
	"strings"
	"sync"
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
var embyRefreshCheckChan = make(chan struct{}, 1)
var embyRefreshCoordinatorOnce sync.Once

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

type DownloadTaskStatusChangedPayload struct {
	TaskId     uint           `json:"task_id"`
	SyncFileId uint           `json:"sync_file_id"`
	Status     DownloadStatus `json:"status"`
	Source     DownloadSource `json:"source"`
}

func TriggerEmbyLibraryRefreshCheck() {
	select {
	case embyRefreshCheckChan <- struct{}{}:
	default:
	}
}

func CountActiveDownloadTasksBySyncPathIds(syncPathIds []uint) (int64, error) {
	if len(syncPathIds) == 0 {
		return 0, nil
	}
	var count int64
	err := db.Db.Model(&DbDownloadTask{}).
		Joins("JOIN sync_files ON sync_files.id = db_download_tasks.sync_file_id").
		Where("sync_files.sync_path_id IN ?", syncPathIds).
		Where(
			"(db_download_tasks.status IN ? OR (db_download_tasks.status = ? AND db_download_tasks.retry_count < ?))",
			[]DownloadStatus{DownloadStatusPending, DownloadStatusDownloading},
			DownloadStatusFailed,
			DefaultQueueRetryMax,
		).
		Count(&count).Error
	return count, err
}

func HasActiveStrmSyncTask(syncPathIds []uint) bool {
	if IsStrmSyncTaskActiveFunc == nil {
		return false
	}
	for _, syncPathId := range syncPathIds {
		if IsStrmSyncTaskActiveFunc(syncPathId) {
			return true
		}
	}
	return false
}

func GetEmbySyncPathIdsByLibraryId(libraryId string) []uint {
	var relations []EmbyLibrarySyncPath
	if err := db.Db.Where("library_id = ?", libraryId).Find(&relations).Error; err != nil {
		helpers.AppLogger.Errorf("查询Emby媒体库 %s 关联同步目录失败: %v", libraryId, err)
		return []uint{}
	}
	ids := make([]uint, 0, len(relations))
	for _, rel := range relations {
		if rel.SyncPathId > 0 {
			ids = append(ids, rel.SyncPathId)
		}
	}
	return mergeSyncPathIds(ids, nil)
}

func IsEmbyLibraryRefreshTaskReady(task *EmbyLibraryRefreshTask, now int64) (bool, string, error) {
	if task == nil {
		return false, "empty_task", nil
	}
	if task.Status != EmbyLibraryRefreshStatusPending {
		return false, "not_pending", nil
	}
	if task.RefreshAfterAt > now && task.DeadlineAt > now {
		return false, "debounce", nil
	}

	syncPathIds := task.GetSyncPathIds()
	if len(syncPathIds) == 0 {
		return false, "empty_sync_paths", nil
	}
	waitSyncPathIds := mergeSyncPathIds(syncPathIds, GetEmbySyncPathIdsByLibraryId(task.LibraryId))

	if task.DeadlineAt > 0 && task.DeadlineAt <= now {
		return true, "deadline", nil
	}

	if HasActiveStrmSyncTask(waitSyncPathIds) {
		return false, "sync_running", nil
	}

	activeDownloads, err := CountActiveDownloadTasksBySyncPathIds(syncPathIds)
	if err != nil {
		return false, "download_query_error", err
	}
	if activeDownloads > 0 {
		return false, "download_running", nil
	}

	return true, "ready", nil
}

func NotifyEmbyRefreshDownloadTaskChanged(syncFileId uint) error {
	if syncFileId == 0 {
		return nil
	}
	var syncFile SyncFile
	if err := db.Db.Select("id", "sync_path_id").First(&syncFile, syncFileId).Error; err != nil {
		return nil
	}
	now := nowUnix()
	libraries := GetEmbyLibraryIdsBySyncPathId(syncFile.SyncPathId)
	for libraryId := range libraries {
		var task EmbyLibraryRefreshTask
		if err := db.Db.Where("library_id = ? AND status = ?", libraryId, EmbyLibraryRefreshStatusPending).First(&task).Error; err != nil {
			continue
		}
		task.LastEventAt = now
		task.RefreshAfterAt = now + DefaultEmbyRefreshDebounceSeconds
		if err := saveEmbyLibraryRefreshTask(&task); err != nil {
			return err
		}
	}
	TriggerEmbyLibraryRefreshCheck()
	return nil
}

func HandleDownloadTaskStatusChanged(event helpers.Event) {
	payload, ok := event.Data.(DownloadTaskStatusChangedPayload)
	if !ok {
		return
	}
	if err := NotifyEmbyRefreshDownloadTaskChanged(payload.SyncFileId); err != nil {
		helpers.AppLogger.Errorf("处理下载任务状态变化事件失败: %v", err)
	}
}

func InitEmbyLibraryRefreshCoordinator() {
	embyRefreshCoordinatorOnce.Do(func() {
		resetRefreshingEmbyLibraryRefreshTasks()
		helpers.Subscribe(helpers.DownloadTaskStatusChangedEvent, HandleDownloadTaskStatusChanged)
		go runEmbyLibraryRefreshScanner()
	})
}

func resetRefreshingEmbyLibraryRefreshTasks() {
	now := nowUnix()
	if err := db.Db.Model(&EmbyLibraryRefreshTask{}).
		Where("status = ?", EmbyLibraryRefreshStatusRefreshing).
		Updates(map[string]interface{}{
			"status":           EmbyLibraryRefreshStatusPending,
			"last_event_at":    now,
			"refresh_after_at": now + DefaultEmbyRefreshDebounceSeconds,
			"error":            "服务重启后重置刷新中任务",
		}).Error; err != nil {
		helpers.AppLogger.Errorf("重置Emby媒体库刷新中任务失败: %v", err)
	}
}

func runEmbyLibraryRefreshScanner() {
	ticker := time.NewTicker(time.Duration(DefaultEmbyRefreshScanSeconds) * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-embyRefreshCheckChan:
			CheckPendingEmbyLibraryRefreshTasks()
		case <-ticker.C:
			CheckPendingEmbyLibraryRefreshTasks()
		}
	}
}

func CheckPendingEmbyLibraryRefreshTasks() {
	if GlobalEmbyConfig == nil || GlobalEmbyConfig.EmbyUrl == "" || GlobalEmbyConfig.EmbyApiKey == "" || GlobalEmbyConfig.EnableRefreshLibrary == 0 {
		helpers.AppLogger.Infof("Emby未配置或未启用刷新媒体库，跳过待刷新任务扫描")
		return
	}

	var tasks []EmbyLibraryRefreshTask
	now := nowUnix()
	if err := db.Db.Where("status = ?", EmbyLibraryRefreshStatusPending).
		Where("refresh_after_at <= ? OR deadline_at <= ?", now, now).
		Order("updated_at ASC").
		Find(&tasks).Error; err != nil {
		helpers.AppLogger.Errorf("查询Emby媒体库待刷新任务失败: %v", err)
		return
	}

	for i := range tasks {
		task := tasks[i]
		ready, reason, err := IsEmbyLibraryRefreshTaskReady(&task, now)
		task.LastCheckedAt = now
		if err != nil {
			task.Error = err.Error()
			saveEmbyLibraryRefreshTask(&task)
			continue
		}
		if !ready {
			saveEmbyLibraryRefreshTask(&task)
			helpers.AppLogger.Debugf("Emby媒体库 %s 暂不刷新，原因: %s", task.LibraryName, reason)
			continue
		}
		if reason == "deadline" {
			helpers.AppLogger.Warnf("Emby媒体库 %s 等待超过最大时长，执行兜底刷新", task.LibraryName)
		}
		if err := refreshEmbyLibraryTask(&task); err != nil {
			helpers.AppLogger.Errorf("刷新Emby媒体库 %s 失败: %v", task.LibraryName, err)
		}
	}
}

func refreshEmbyLibraryTask(task *EmbyLibraryRefreshTask) error {
	if GlobalEmbyConfig == nil || GlobalEmbyConfig.EmbyUrl == "" || GlobalEmbyConfig.EmbyApiKey == "" || GlobalEmbyConfig.EnableRefreshLibrary == 0 {
		return nil
	}
	now := nowUnix()
	result := db.Db.Model(&EmbyLibraryRefreshTask{}).
		Where("id = ? AND status = ?", task.ID, EmbyLibraryRefreshStatusPending).
		Updates(map[string]interface{}{
			"status":          EmbyLibraryRefreshStatusRefreshing,
			"last_checked_at": now,
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return nil
	}
	task.Status = EmbyLibraryRefreshStatusRefreshing
	task.LastCheckedAt = now

	client := embyclientrestgo.NewClient(GlobalEmbyConfig.EmbyUrl, GlobalEmbyConfig.EmbyApiKey)
	if err := client.RefreshLibrary(task.LibraryId, task.LibraryName); err != nil {
		task.Status = EmbyLibraryRefreshStatusFailed
		task.Error = err.Error()
		saveEmbyLibraryRefreshTask(task)
		return err
	}
	return markEmbyRefreshTaskCompleted(task)
}

func markEmbyRefreshTaskCompleted(task *EmbyLibraryRefreshTask) error {
	task.Status = EmbyLibraryRefreshStatusCompleted
	task.LastRefreshAt = nowUnix()
	task.Error = ""
	return saveEmbyLibraryRefreshTask(task)
}

func nowUnix() int64 {
	return time.Now().Unix()
}
