package models

import (
	"encoding/json"
	"errors"
	"sort"
	"strings"
	"sync"
	"time"

	"gorm.io/gorm"

	"qmediasync/internal/db"
	embyclientrestgo "qmediasync/internal/embyclient-rest-go"
	"qmediasync/internal/helpers"
)

const (
	EmbyLibraryRefreshTargetTypeLibrary = "library"
	EmbyLibraryRefreshTargetTypeItem    = "item"

	EmbyLibraryRefreshStatusPending    = "pending"
	EmbyLibraryRefreshStatusRefreshing = "refreshing"
	EmbyLibraryRefreshStatusCompleted  = "completed"
	EmbyLibraryRefreshStatusFailed     = "failed"
	EmbyLibraryRefreshStatusCancelled  = "cancelled"

	DefaultEmbyRefreshDebounceSeconds           = int64(10)
	DefaultEmbyRefreshMaxWaitSeconds            = int64(6 * 60 * 60)
	DefaultEmbyRefreshScanSeconds               = int64(60)
	DefaultEmbyRefreshDownloadEventBatchSeconds = int64(5)
)

var IsStrmSyncTaskActiveFunc func(syncPathId uint) bool
var embyRefreshCheckChan = make(chan struct{}, 1)
var embyRefreshCoordinatorOnce sync.Once
var embyRefreshDownloadEventBatch = &downloadEventBatch{
	syncPathIds: make(map[uint]struct{}),
	syncFileIds: make(map[uint]struct{}),
}
var embyRefreshScannerConfigState = struct {
	sync.Mutex
	initialized bool
	enabled     bool
}{}
var embyRefreshTimerState = struct {
	sync.Mutex
	timer       *time.Timer
	nextCheckAt int64
	generation  uint64
}{}

type downloadEventBatch struct {
	mutex       sync.Mutex
	syncPathIds map[uint]struct{}
	syncFileIds map[uint]struct{}
}

type EmbyLibraryRefreshTask struct {
	BaseModel
	LibraryId           string `json:"library_id" gorm:"uniqueIndex;type:varchar(128)"`
	LibraryName         string `json:"library_name" gorm:"type:varchar(255)"`
	SyncPathIdsStr      string `json:"-" gorm:"type:text;default:'[]'"`
	TargetType          string `json:"target_type" gorm:"type:varchar(32);index;default:library"`
	ItemIdsStr          string `json:"-" gorm:"type:text;default:'[]'"`
	ItemRecursive       bool   `json:"item_recursive" gorm:"default:false"`
	FallbackLibraryId   string `json:"fallback_library_id" gorm:"type:varchar(128);index"`
	FallbackLibraryName string `json:"fallback_library_name" gorm:"type:varchar(255)"`
	Status              string `json:"status" gorm:"index;type:varchar(32)"`
	LastEventAt         int64  `json:"last_event_at" gorm:"index"`
	RefreshAfterAt      int64  `json:"refresh_after_at" gorm:"index"`
	DeadlineAt          int64  `json:"deadline_at" gorm:"index"`
	LastCheckedAt       int64  `json:"last_checked_at"`
	LastRefreshAt       int64  `json:"last_refresh_at"`
	Error               string `json:"error" gorm:"type:text"`
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
		helpers.AppLogger.Warnf("解析 Emby 媒体库刷新任务 sync_path_ids 失败：%v", err)
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

func (t *EmbyLibraryRefreshTask) GetItemIds() []string {
	var ids []string
	if t == nil || t.ItemIdsStr == "" {
		return ids
	}
	if err := json.Unmarshal([]byte(t.ItemIdsStr), &ids); err != nil {
		helpers.AppLogger.Warnf("解析 Emby 条目刷新任务 item_ids 失败：%v", err)
		return []string{}
	}
	sort.Strings(ids)
	return ids
}

func (t *EmbyLibraryRefreshTask) SetItemIds(ids []string) {
	merged := mergeStringIds(ids, nil)
	data, err := json.Marshal(merged)
	if err != nil {
		t.ItemIdsStr = "[]"
		return
	}
	t.ItemIdsStr = string(data)
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

func mergeStringIds(left []string, right []string) []string {
	seen := make(map[string]bool)
	for _, id := range left {
		if id != "" {
			seen[id] = true
		}
	}
	for _, id := range right {
		if id != "" {
			seen[id] = true
		}
	}
	ids := make([]string, 0, len(seen))
	for id := range seen {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

func newPendingEmbyLibraryRefreshTask(libraryId string, libraryName string, syncPathIds []uint, now int64) *EmbyLibraryRefreshTask {
	task := &EmbyLibraryRefreshTask{
		LibraryId:      libraryId,
		LibraryName:    libraryName,
		TargetType:     EmbyLibraryRefreshTargetTypeLibrary,
		Status:         EmbyLibraryRefreshStatusPending,
		LastEventAt:    now,
		RefreshAfterAt: now + DefaultEmbyRefreshDebounceSeconds,
		DeadlineAt:     now + DefaultEmbyRefreshMaxWaitSeconds,
	}
	task.SetSyncPathIds(syncPathIds)
	return task
}

func isEmbyLibraryRefreshEnabled() bool {
	return GlobalEmbyConfig != nil &&
		GlobalEmbyConfig.EmbyUrl != "" &&
		GlobalEmbyConfig.EmbyApiKey != "" &&
		GlobalEmbyConfig.EnableRefreshLibrary != 0
}

func markEmbyRefreshScannerConfigState(enabled bool) bool {
	embyRefreshScannerConfigState.Lock()
	defer embyRefreshScannerConfigState.Unlock()

	shouldLogDisabledTransition := embyRefreshScannerConfigState.initialized &&
		embyRefreshScannerConfigState.enabled &&
		!enabled
	embyRefreshScannerConfigState.initialized = true
	embyRefreshScannerConfigState.enabled = enabled
	return shouldLogDisabledTransition
}

func resetEmbyRefreshScannerConfigStateForTest() {
	embyRefreshScannerConfigState.Lock()
	defer embyRefreshScannerConfigState.Unlock()

	embyRefreshScannerConfigState.initialized = false
	embyRefreshScannerConfigState.enabled = false
}

func resetEmbyRefreshTimerStateForTest() {
	embyRefreshTimerState.Lock()
	defer embyRefreshTimerState.Unlock()

	if embyRefreshTimerState.timer != nil {
		embyRefreshTimerState.timer.Stop()
	}
	embyRefreshTimerState.timer = nil
	embyRefreshTimerState.nextCheckAt = 0
	embyRefreshTimerState.generation++
}

func saveEmbyLibraryRefreshTask(task *EmbyLibraryRefreshTask) error {
	return db.Db.Save(task).Error
}

func RequestEmbyLibraryRefreshBySyncPathId(syncPathId uint) error {
	if syncPathId == 0 {
		helpers.AppLogger.Infof("临时同步路径不触发 Emby 媒体库刷新")
		return nil
	}
	if GlobalEmbyConfig == nil || GlobalEmbyConfig.EmbyUrl == "" || GlobalEmbyConfig.EmbyApiKey == "" || GlobalEmbyConfig.EnableRefreshLibrary == 0 {
		helpers.AppLogger.Infof("Emby 未配置或未启用刷新媒体库，跳过提交刷新任务")
		return nil
	}

	libraries := GetEmbyLibraryIdsBySyncPathId(syncPathId)
	if len(libraries) == 0 {
		helpers.AppLogger.Infof("同步目录 %d 未关联 Emby 媒体库，跳过提交刷新任务", syncPathId)
		return nil
	}

	now := nowUnix()
	for libraryId, libraryName := range libraries {
		if err := upsertEmbyLibraryRefreshTask(libraryId, libraryName, syncPathId, now); err != nil {
			return err
		}
	}
	ScheduleNextEmbyLibraryRefreshCheck()
	TriggerEmbyLibraryRefreshCheck()
	return nil
}

// RequestEmbyRefreshBySyncFile 根据 STRM 对应的 SyncFile 提交 Emby 刷新任务。
func RequestEmbyRefreshBySyncFile(syncFile *SyncFile) error {
	if syncFile == nil {
		return nil
	}
	if !isEmbyLibraryRefreshEnabled() {
		helpers.AppLogger.Infof("Emby 未配置或未启用刷新媒体库，跳过提交刷新任务")
		return nil
	}
	target, err := ResolveEmbyRefreshTarget(syncFile)
	if err != nil {
		return err
	}
	if target.TargetType != EmbyRefreshTargetTypeItem || target.ItemID == "" {
		return RequestEmbyLibraryRefreshBySyncPathId(syncFile.SyncPathId)
	}
	if err := upsertEmbyItemRefreshTask(target, syncFile.SyncPathId, nowUnix()); err != nil {
		return err
	}
	ScheduleNextEmbyLibraryRefreshCheck()
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
	task.TargetType = EmbyLibraryRefreshTargetTypeLibrary
	task.Status = EmbyLibraryRefreshStatusPending
	task.LastEventAt = now
	task.RefreshAfterAt = now + DefaultEmbyRefreshDebounceSeconds
	if len(mergedIds) > len(existingIds) || task.DeadlineAt <= now || oldStatus == EmbyLibraryRefreshStatusCompleted || oldStatus == EmbyLibraryRefreshStatusFailed || oldStatus == EmbyLibraryRefreshStatusCancelled {
		task.DeadlineAt = now + DefaultEmbyRefreshMaxWaitSeconds
	}
	task.LastCheckedAt = 0
	task.Error = ""
	task.SetSyncPathIds(mergedIds)
	return tx.Save(&task).Error
}

func newPendingEmbyItemRefreshTask(target EmbyRefreshTarget, syncPathId uint, now int64) *EmbyLibraryRefreshTask {
	task := &EmbyLibraryRefreshTask{
		LibraryId:           embyItemRefreshTaskKey(target.ItemID),
		LibraryName:         target.ItemName,
		TargetType:          EmbyLibraryRefreshTargetTypeItem,
		ItemRecursive:       target.Recursive,
		FallbackLibraryId:   target.FallbackLibraryId,
		FallbackLibraryName: target.FallbackLibraryName,
		Status:              EmbyLibraryRefreshStatusPending,
		LastEventAt:         now,
		RefreshAfterAt:      now + DefaultEmbyRefreshDebounceSeconds,
		DeadlineAt:          now + DefaultEmbyRefreshMaxWaitSeconds,
	}
	task.SetSyncPathIds([]uint{syncPathId})
	task.SetItemIds([]string{target.ItemID})
	return task
}

func embyItemRefreshTaskKey(itemID string) string {
	return "item:" + itemID
}

func upsertEmbyItemRefreshTask(target EmbyRefreshTarget, syncPathId uint, now int64) error {
	return db.Db.Transaction(func(tx *gorm.DB) error {
		return upsertEmbyItemRefreshTaskWithDB(tx, target, syncPathId, now)
	})
}

func upsertEmbyItemRefreshTaskWithDB(tx *gorm.DB, target EmbyRefreshTarget, syncPathId uint, now int64) error {
	key := embyItemRefreshTaskKey(target.ItemID)
	var task EmbyLibraryRefreshTask
	err := tx.Where("library_id = ?", key).First(&task).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		task = *newPendingEmbyItemRefreshTask(target, syncPathId, now)
		if err := tx.Create(&task).Error; err != nil {
			if isUniqueConstraintError(err) {
				return upsertEmbyItemRefreshTaskWithDB(tx, target, syncPathId, now)
			}
			return err
		}
		return nil
	}
	if err != nil {
		return err
	}

	existingSyncPathIds := task.GetSyncPathIds()
	mergedSyncPathIds := mergeSyncPathIds(existingSyncPathIds, []uint{syncPathId})
	existingItemIds := task.GetItemIds()
	mergedItemIds := mergeStringIds(existingItemIds, []string{target.ItemID})
	oldStatus := task.Status
	task.LibraryName = target.ItemName
	task.TargetType = EmbyLibraryRefreshTargetTypeItem
	task.ItemRecursive = target.Recursive
	task.FallbackLibraryId = target.FallbackLibraryId
	task.FallbackLibraryName = target.FallbackLibraryName
	task.Status = EmbyLibraryRefreshStatusPending
	task.LastEventAt = now
	task.RefreshAfterAt = now + DefaultEmbyRefreshDebounceSeconds
	if len(mergedSyncPathIds) > len(existingSyncPathIds) ||
		len(mergedItemIds) > len(existingItemIds) ||
		task.DeadlineAt <= now ||
		oldStatus == EmbyLibraryRefreshStatusCompleted ||
		oldStatus == EmbyLibraryRefreshStatusFailed ||
		oldStatus == EmbyLibraryRefreshStatusCancelled {
		task.DeadlineAt = now + DefaultEmbyRefreshMaxWaitSeconds
	}
	task.LastCheckedAt = 0
	task.Error = ""
	task.SetSyncPathIds(mergedSyncPathIds)
	task.SetItemIds(mergedItemIds)
	return tx.Save(&task).Error
}

func cancelPendingEmbyLibraryRefreshTasksBySyncPathIdsWithDB(tx *gorm.DB, syncPathIds []uint, reason string) error {
	syncPathIds = mergeSyncPathIds(syncPathIds, nil)
	if len(syncPathIds) == 0 {
		return nil
	}

	var relations []EmbyLibrarySyncPath
	if err := tx.Where("sync_path_id IN ?", syncPathIds).Find(&relations).Error; err != nil {
		return err
	}
	libraryIdSet := make(map[string]struct{})
	for _, relation := range relations {
		if relation.LibraryId != "" {
			libraryIdSet[relation.LibraryId] = struct{}{}
		}
	}
	libraryIds := make([]string, 0, len(libraryIdSet))
	for libraryId := range libraryIdSet {
		libraryIds = append(libraryIds, libraryId)
	}
	if len(libraryIds) == 0 {
		return nil
	}

	now := nowUnix()
	return tx.Model(&EmbyLibraryRefreshTask{}).
		Where("library_id IN ? AND status = ?", libraryIds, EmbyLibraryRefreshStatusPending).
		Updates(map[string]interface{}{
			"status":          EmbyLibraryRefreshStatusCancelled,
			"last_checked_at": now,
			"error":           reason,
		}).Error
}

// CancelPendingEmbyLibraryRefreshTasksBySyncPathIds 按同步目录取消待执行的 Emby 媒体库刷新任务。
func CancelPendingEmbyLibraryRefreshTasksBySyncPathIds(syncPathIds []uint, reason string) error {
	return cancelPendingEmbyLibraryRefreshTasksBySyncPathIdsWithDB(db.Db, syncPathIds, reason)
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
	SyncPathId uint           `json:"sync_path_id"`
	Status     DownloadStatus `json:"status"`
	Source     DownloadSource `json:"source"`
}

func TriggerEmbyLibraryRefreshCheck() {
	select {
	case embyRefreshCheckChan <- struct{}{}:
	default:
	}
}

func nextPendingEmbyLibraryRefreshCheckAt(now int64) (int64, bool, error) {
	var task EmbyLibraryRefreshTask
	err := db.Db.Select("refresh_after_at").
		Where("status = ?", EmbyLibraryRefreshStatusPending).
		Where("refresh_after_at > ?", now).
		Order("refresh_after_at ASC").
		First(&task).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, err
	}
	return task.RefreshAfterAt, true, nil
}

func hasDueUncheckedEmbyLibraryRefreshTask(now int64) (bool, error) {
	var count int64
	err := db.Db.Model(&EmbyLibraryRefreshTask{}).
		Where("status = ?", EmbyLibraryRefreshStatusPending).
		Where("refresh_after_at <= ?", now).
		Where("(last_checked_at = 0 OR last_checked_at < refresh_after_at)").
		Count(&count).Error
	return count > 0, err
}

func setNextEmbyLibraryRefreshCheckTimer(nextCheckAt int64, hasNext bool) {
	embyRefreshTimerState.Lock()
	defer embyRefreshTimerState.Unlock()

	// 并发调度时，较旧的 DB 查询结果可能晚于较新的更早结果返回。
	// 保留已有更早 timer，最多提前检查一次。
	if embyRefreshTimerState.timer != nil && embyRefreshTimerState.nextCheckAt > 0 {
		if !hasNext || embyRefreshTimerState.nextCheckAt <= nextCheckAt {
			return
		}
	}

	if !hasNext {
		if embyRefreshTimerState.timer != nil {
			embyRefreshTimerState.timer.Stop()
		}
		embyRefreshTimerState.timer = nil
		embyRefreshTimerState.nextCheckAt = 0
		embyRefreshTimerState.generation++
		return
	}

	if embyRefreshTimerState.timer != nil && embyRefreshTimerState.nextCheckAt == nextCheckAt {
		return
	}
	if embyRefreshTimerState.timer != nil {
		embyRefreshTimerState.timer.Stop()
	}

	delay := time.Until(time.Unix(nextCheckAt, 0))
	if delay < 0 {
		delay = 0
	}
	scheduledAt := nextCheckAt
	embyRefreshTimerState.generation++
	generation := embyRefreshTimerState.generation
	embyRefreshTimerState.timer = time.AfterFunc(delay, func() {
		shouldTrigger := false
		embyRefreshTimerState.Lock()
		if embyRefreshTimerState.nextCheckAt == scheduledAt && embyRefreshTimerState.generation == generation {
			embyRefreshTimerState.timer = nil
			embyRefreshTimerState.nextCheckAt = 0
			embyRefreshTimerState.generation++
			shouldTrigger = true
		}
		embyRefreshTimerState.Unlock()
		if shouldTrigger {
			TriggerEmbyLibraryRefreshCheck()
		}
	})
	embyRefreshTimerState.nextCheckAt = nextCheckAt
}

// ScheduleNextEmbyLibraryRefreshCheck 调度最近一个未来的 Emby 媒体库刷新检查。
func ScheduleNextEmbyLibraryRefreshCheck() {
	if db.Db == nil {
		return
	}
	now := nowUnix()
	dueUnchecked, err := hasDueUncheckedEmbyLibraryRefreshTask(now)
	if err != nil {
		helpers.AppLogger.Errorf("查询已到期 Emby 媒体库刷新任务失败：%v", err)
		return
	}
	nextCheckAt, hasNext, err := nextPendingEmbyLibraryRefreshCheckAt(now)
	if err != nil {
		helpers.AppLogger.Errorf("调度下一次 Emby 媒体库刷新检查失败：%v", err)
		return
	}
	setNextEmbyLibraryRefreshCheckTimer(nextCheckAt, hasNext)
	if dueUnchecked {
		TriggerEmbyLibraryRefreshCheck()
	}
}

func CountActiveDownloadTasksBySyncPathIds(syncPathIds []uint) (int64, error) {
	if len(syncPathIds) == 0 {
		return 0, nil
	}
	var count int64
	err := db.Db.Model(&DbDownloadTask{}).
		Joins("LEFT JOIN sync_files ON sync_files.id = db_download_tasks.sync_file_id").
		Where(
			"(db_download_tasks.sync_path_id IN ? OR ((db_download_tasks.sync_path_id = 0 OR db_download_tasks.sync_path_id IS NULL) AND sync_files.sync_path_id IN ?))",
			syncPathIds,
			syncPathIds,
		).
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
		helpers.AppLogger.Errorf("查询 Emby 媒体库 %s 关联同步目录失败：%v", libraryId, err)
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
	if task.DeadlineAt > 0 && task.DeadlineAt <= now {
		return false, "deadline_expired", nil
	}
	if task.RefreshAfterAt > now && task.DeadlineAt > now {
		return false, "debounce", nil
	}

	syncPathIds := task.GetSyncPathIds()
	if len(syncPathIds) == 0 {
		return false, "empty_sync_paths", nil
	}
	waitSyncPathIds := mergeSyncPathIds(syncPathIds, GetEmbySyncPathIdsByLibraryId(task.LibraryId))

	if HasActiveStrmSyncTask(waitSyncPathIds) {
		return false, "sync_running", nil
	}

	activeDownloads, err := CountActiveDownloadTasksBySyncPathIds(waitSyncPathIds)
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
	return NotifyEmbyRefreshDownloadTasksChanged([]uint{syncFileId})
}

func NotifyEmbyRefreshDownloadTasksChanged(syncFileIds []uint) error {
	syncFileIds = uniqueUintIds(syncFileIds)
	if len(syncFileIds) == 0 {
		return nil
	}

	var syncFiles []SyncFile
	if err := db.Db.Select("id", "sync_path_id").Where("id IN ?", syncFileIds).Find(&syncFiles).Error; err != nil {
		return err
	}
	syncPathIds := make([]uint, 0, len(syncFiles))
	for _, syncFile := range syncFiles {
		if syncFile.SyncPathId > 0 {
			syncPathIds = append(syncPathIds, syncFile.SyncPathId)
		}
	}
	syncPathIds = mergeSyncPathIds(syncPathIds, nil)
	return NotifyEmbyRefreshDownloadTasksChangedBySyncPathIds(syncPathIds)
}

func NotifyEmbyRefreshDownloadTasksChangedBySyncPathIds(syncPathIds []uint) error {
	syncPathIds = mergeSyncPathIds(syncPathIds, nil)
	if len(syncPathIds) == 0 {
		return nil
	}

	var relations []EmbyLibrarySyncPath
	if err := db.Db.Where("sync_path_id IN ?", syncPathIds).Find(&relations).Error; err != nil {
		return err
	}
	libraryIdSet := make(map[string]struct{})
	for _, relation := range relations {
		if relation.LibraryId != "" {
			libraryIdSet[relation.LibraryId] = struct{}{}
		}
	}
	libraryIds := make([]string, 0, len(libraryIdSet))
	for libraryId := range libraryIdSet {
		libraryIds = append(libraryIds, libraryId)
	}
	if len(libraryIds) == 0 {
		return nil
	}

	now := nowUnix()
	if err := db.Db.Model(&EmbyLibraryRefreshTask{}).
		Where("library_id IN ? AND status = ?", libraryIds, EmbyLibraryRefreshStatusPending).
		Updates(map[string]interface{}{
			"last_event_at":    now,
			"refresh_after_at": now + DefaultEmbyRefreshDebounceSeconds,
		}).Error; err != nil {
		return err
	}
	ScheduleNextEmbyLibraryRefreshCheck()
	TriggerEmbyLibraryRefreshCheck()
	return nil
}

func HandleDownloadTaskStatusChanged(event helpers.Event) {
	payload, ok := event.Data.(DownloadTaskStatusChangedPayload)
	if !ok {
		return
	}
	enqueueEmbyRefreshDownloadTaskChanged(payload.SyncPathId, payload.SyncFileId)
}

func enqueueEmbyRefreshDownloadTaskChanged(syncPathId uint, syncFileId uint) {
	embyRefreshDownloadEventBatch.mutex.Lock()
	defer embyRefreshDownloadEventBatch.mutex.Unlock()
	if syncPathId > 0 {
		embyRefreshDownloadEventBatch.syncPathIds[syncPathId] = struct{}{}
		return
	}
	if syncFileId == 0 {
		return
	}
	embyRefreshDownloadEventBatch.syncFileIds[syncFileId] = struct{}{}
}

func drainPendingEmbyRefreshDownloadTaskChanges() ([]uint, []uint) {
	embyRefreshDownloadEventBatch.mutex.Lock()
	defer embyRefreshDownloadEventBatch.mutex.Unlock()
	if len(embyRefreshDownloadEventBatch.syncPathIds) == 0 && len(embyRefreshDownloadEventBatch.syncFileIds) == 0 {
		return nil, nil
	}
	syncPathIds := make([]uint, 0, len(embyRefreshDownloadEventBatch.syncPathIds))
	for syncPathId := range embyRefreshDownloadEventBatch.syncPathIds {
		syncPathIds = append(syncPathIds, syncPathId)
	}
	syncFileIds := make([]uint, 0, len(embyRefreshDownloadEventBatch.syncFileIds))
	for syncFileId := range embyRefreshDownloadEventBatch.syncFileIds {
		syncFileIds = append(syncFileIds, syncFileId)
	}
	embyRefreshDownloadEventBatch.syncPathIds = make(map[uint]struct{})
	embyRefreshDownloadEventBatch.syncFileIds = make(map[uint]struct{})
	return syncPathIds, syncFileIds
}

func flushPendingEmbyRefreshDownloadTaskChanges() error {
	syncPathIds, syncFileIds := drainPendingEmbyRefreshDownloadTaskChanges()
	if len(syncPathIds) > 0 {
		if err := NotifyEmbyRefreshDownloadTasksChangedBySyncPathIds(syncPathIds); err != nil {
			return err
		}
	}
	if len(syncFileIds) > 0 {
		return NotifyEmbyRefreshDownloadTasksChanged(syncFileIds)
	}
	return nil
}

func runEmbyLibraryRefreshDownloadEventBatcher() {
	ticker := time.NewTicker(time.Duration(DefaultEmbyRefreshDownloadEventBatchSeconds) * time.Second)
	defer ticker.Stop()
	for range ticker.C {
		if err := flushPendingEmbyRefreshDownloadTaskChanges(); err != nil {
			helpers.AppLogger.Errorf("批量处理下载任务状态变化事件失败：%v", err)
		}
	}
}

func InitEmbyLibraryRefreshCoordinator() {
	embyRefreshCoordinatorOnce.Do(func() {
		resetRefreshingEmbyLibraryRefreshTasks()
		helpers.Subscribe(helpers.DownloadTaskStatusChangedEvent, HandleDownloadTaskStatusChanged)
		go runEmbyLibraryRefreshDownloadEventBatcher()
		go runEmbyLibraryRefreshScanner()
		ScheduleNextEmbyLibraryRefreshCheck()
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
		helpers.AppLogger.Errorf("重置 Emby 媒体库刷新中任务失败：%v", err)
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
	if !isEmbyLibraryRefreshEnabled() {
		if markEmbyRefreshScannerConfigState(false) {
			helpers.AppLogger.Infof("Emby 未配置或未启用刷新媒体库，暂停待刷新任务扫描")
		}
		return
	}
	markEmbyRefreshScannerConfigState(true)
	defer ScheduleNextEmbyLibraryRefreshCheck()
	if err := flushPendingEmbyRefreshDownloadTaskChanges(); err != nil {
		helpers.AppLogger.Errorf("扫描 Emby 媒体库刷新任务前批量处理下载事件失败：%v", err)
	}

	var tasks []EmbyLibraryRefreshTask
	now := nowUnix()
	if err := db.Db.Where("status = ?", EmbyLibraryRefreshStatusPending).
		Where("refresh_after_at <= ? OR deadline_at <= ?", now, now).
		Order("updated_at ASC").
		Find(&tasks).Error; err != nil {
		helpers.AppLogger.Errorf("查询 Emby 媒体库待刷新任务失败：%v", err)
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
		if reason == "deadline_expired" {
			if err := markEmbyRefreshTaskCancelled(&task, "等待超过最大时长，取消刷新"); err != nil {
				helpers.AppLogger.Errorf("取消 Emby 媒体库 %s 刷新任务失败：%v", task.LibraryName, err)
			}
			helpers.AppLogger.Warnf("Emby 媒体库 %s 等待超过最大时长，取消刷新", task.LibraryName)
			continue
		}
		if !ready {
			saveEmbyLibraryRefreshTask(&task)
			helpers.AppLogger.Debugf("Emby 媒体库 %s 暂不刷新，原因：%s", task.LibraryName, reason)
			continue
		}
		if err := refreshEmbyLibraryTask(&task); err != nil {
			helpers.AppLogger.Errorf("刷新 Emby 媒体库 %s 失败：%v", task.LibraryName, err)
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
	if err := executeEmbyRefreshTask(client, task); err != nil {
		task.Status = EmbyLibraryRefreshStatusFailed
		task.Error = err.Error()
		saveEmbyLibraryRefreshTask(task)
		return err
	}
	return markEmbyRefreshTaskCompleted(task)
}

func executeEmbyRefreshTask(client *embyclientrestgo.Client, task *EmbyLibraryRefreshTask) error {
	if task.TargetType == EmbyLibraryRefreshTargetTypeItem {
		itemIds := task.GetItemIds()
		if len(itemIds) == 0 && task.FallbackLibraryId != "" {
			return client.RefreshLibrary(task.FallbackLibraryId, task.FallbackLibraryName)
		}
		for _, itemId := range itemIds {
			if err := client.RefreshItem(itemId, task.LibraryName, task.ItemRecursive); err != nil {
				if task.FallbackLibraryId != "" {
					return client.RefreshLibrary(task.FallbackLibraryId, task.FallbackLibraryName)
				}
				return err
			}
		}
		return nil
	}
	return client.RefreshLibrary(task.LibraryId, task.LibraryName)
}

func markEmbyRefreshTaskCompleted(task *EmbyLibraryRefreshTask) error {
	task.Status = EmbyLibraryRefreshStatusCompleted
	task.LastRefreshAt = nowUnix()
	task.Error = ""
	return saveEmbyLibraryRefreshTask(task)
}

func markEmbyRefreshTaskCancelled(task *EmbyLibraryRefreshTask, reason string) error {
	task.Status = EmbyLibraryRefreshStatusCancelled
	task.LastCheckedAt = nowUnix()
	task.Error = reason
	return saveEmbyLibraryRefreshTask(task)
}

func uniqueUintIds(ids []uint) []uint {
	return mergeSyncPathIds(ids, nil)
}

func nowUnix() int64 {
	return time.Now().Unix()
}
