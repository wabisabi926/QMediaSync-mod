package models

import (
	"bytes"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"

	"qmediasync/internal/db"
	"qmediasync/internal/helpers"
)

func setupEmbyRefreshTestDB(t *testing.T) {
	t.Helper()
	resetEmbyRefreshTimerStateForTest()
	drainEmbyRefreshCheckChan()
	t.Cleanup(resetEmbyRefreshTimerStateForTest)
	t.Cleanup(drainEmbyRefreshCheckChan)
	if helpers.AppLogger == nil {
		helpers.AppLogger = &helpers.QLogger{
			Logger: log.New(io.Discard, "", 0),
		}
	}
	testDb, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("打开测试数据库失败: %v", err)
	}
	db.Db = testDb
	if err := db.Db.AutoMigrate(
		&EmbyConfig{},
		&EmbyLibrarySyncPath{},
		&EmbyLibraryRefreshTask{},
		&SyncFile{},
		&DbDownloadTask{},
	); err != nil {
		t.Fatalf("迁移测试表失败: %v", err)
	}
	GlobalEmbyConfig = &EmbyConfig{
		EmbyUrl:              "http://emby.local:8096",
		EmbyApiKey:           "test-key",
		EnableRefreshLibrary: 1,
	}
	IsStrmSyncTaskActiveFunc = nil
	embyRefreshDownloadEventBatch.mutex.Lock()
	embyRefreshDownloadEventBatch.syncPathIds = make(map[uint]struct{})
	embyRefreshDownloadEventBatch.syncFileIds = make(map[uint]struct{})
	embyRefreshDownloadEventBatch.mutex.Unlock()
}

func drainEmbyRefreshCheckChan() {
	for {
		select {
		case <-embyRefreshCheckChan:
		default:
			return
		}
	}
}

func assertScheduledEmbyRefreshCheckAt(t *testing.T, want int64) {
	t.Helper()
	embyRefreshTimerState.Lock()
	defer embyRefreshTimerState.Unlock()

	if embyRefreshTimerState.timer == nil {
		t.Fatalf("Emby 刷新调度 timer 未设置，期望 nextCheckAt=%d", want)
	}
	if embyRefreshTimerState.nextCheckAt != want {
		t.Fatalf("Emby 刷新 nextCheckAt=%d，期望 %d", embyRefreshTimerState.nextCheckAt, want)
	}
}

func assertNoScheduledEmbyRefreshCheck(t *testing.T) {
	t.Helper()
	embyRefreshTimerState.Lock()
	defer embyRefreshTimerState.Unlock()

	if embyRefreshTimerState.timer != nil || embyRefreshTimerState.nextCheckAt != 0 {
		t.Fatalf("不应设置 Emby 刷新调度 timer，timer=%v nextCheckAt=%d", embyRefreshTimerState.timer, embyRefreshTimerState.nextCheckAt)
	}
}

func TestMergeSyncPathIds(t *testing.T) {
	got := mergeSyncPathIds([]uint{3, 1, 3}, []uint{2, 1})
	want := []uint{1, 2, 3}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("合并后的 sync_path_id = %v，期望 %v", got, want)
	}
}

func TestEmbyLibraryRefreshTaskSyncPathIds(t *testing.T) {
	task := &EmbyLibraryRefreshTask{}
	task.SetSyncPathIds([]uint{5, 2})
	got := task.GetSyncPathIds()
	want := []uint{2, 5}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("解析 sync_path_ids = %v，期望 %v", got, want)
	}
	if task.SyncPathIdsStr != "[2,5]" {
		t.Fatalf("sync_path_ids 字符串 = %s，期望 [2,5]", task.SyncPathIdsStr)
	}
}

func TestRefreshTaskDefaultTimings(t *testing.T) {
	now := time.Now().Unix()
	task := newPendingEmbyLibraryRefreshTask("lib-1", "电影", []uint{1}, now)
	if task.RefreshAfterAt != now+DefaultEmbyRefreshDebounceSeconds {
		t.Fatalf("refresh_after_at = %d，期望 %d", task.RefreshAfterAt, now+DefaultEmbyRefreshDebounceSeconds)
	}
	if task.DeadlineAt != now+DefaultEmbyRefreshMaxWaitSeconds {
		t.Fatalf("deadline_at = %d，期望 %d", task.DeadlineAt, now+DefaultEmbyRefreshMaxWaitSeconds)
	}
	if DefaultEmbyRefreshDownloadEventBatchSeconds != 5 {
		t.Fatalf("下载事件批量处理间隔 = %d，期望 5", DefaultEmbyRefreshDownloadEventBatchSeconds)
	}
}

func TestRequestEmbyLibraryRefreshBySyncPathCreatesTask(t *testing.T) {
	setupEmbyRefreshTestDB(t)
	db.Db.Create(&EmbyLibrarySyncPath{LibraryId: "lib-movie", LibraryName: "电影", SyncPathId: 10})

	if err := RequestEmbyLibraryRefreshBySyncPathId(10); err != nil {
		t.Fatalf("提交刷新任务失败: %v", err)
	}

	var task EmbyLibraryRefreshTask
	if err := db.Db.Where("library_id = ?", "lib-movie").First(&task).Error; err != nil {
		t.Fatalf("查询刷新任务失败: %v", err)
	}
	if task.Status != EmbyLibraryRefreshStatusPending {
		t.Fatalf("刷新任务状态 = %s，期望 pending", task.Status)
	}
	if !reflect.DeepEqual(task.GetSyncPathIds(), []uint{10}) {
		t.Fatalf("sync_path_ids = %v，期望 [10]", task.GetSyncPathIds())
	}
}

func TestRequestEmbyLibraryRefreshMergesSameLibrary(t *testing.T) {
	setupEmbyRefreshTestDB(t)
	db.Db.Create(&EmbyLibrarySyncPath{LibraryId: "lib-movie", LibraryName: "电影", SyncPathId: 10})
	db.Db.Create(&EmbyLibrarySyncPath{LibraryId: "lib-movie", LibraryName: "电影", SyncPathId: 11})

	if err := RequestEmbyLibraryRefreshBySyncPathId(10); err != nil {
		t.Fatalf("第一次提交刷新任务失败: %v", err)
	}
	if err := RequestEmbyLibraryRefreshBySyncPathId(11); err != nil {
		t.Fatalf("第二次提交刷新任务失败: %v", err)
	}

	var tasks []EmbyLibraryRefreshTask
	db.Db.Find(&tasks)
	if len(tasks) != 1 {
		t.Fatalf("同一媒体库应合并为 1 条任务，实际 %d", len(tasks))
	}
	if !reflect.DeepEqual(tasks[0].GetSyncPathIds(), []uint{10, 11}) {
		t.Fatalf("sync_path_ids = %v，期望 [10 11]", tasks[0].GetSyncPathIds())
	}
}

func TestRequestEmbyLibraryRefreshSkipsDisabledConfig(t *testing.T) {
	setupEmbyRefreshTestDB(t)
	GlobalEmbyConfig.EnableRefreshLibrary = 0
	db.Db.Create(&EmbyLibrarySyncPath{LibraryId: "lib-movie", LibraryName: "电影", SyncPathId: 10})

	if err := RequestEmbyLibraryRefreshBySyncPathId(10); err != nil {
		t.Fatalf("关闭刷新时应安静跳过，实际错误: %v", err)
	}

	var total int64
	db.Db.Model(&EmbyLibraryRefreshTask{}).Count(&total)
	if total != 0 {
		t.Fatalf("关闭刷新时不应创建任务，实际 %d", total)
	}
}

func setEmbyRefreshTestLogger(t *testing.T) *bytes.Buffer {
	t.Helper()
	var buf bytes.Buffer
	helpers.AppLogger = &helpers.QLogger{
		Logger: log.New(&buf, "", 0),
	}
	return &buf
}

func TestCheckPendingEmbyLibraryRefreshTasksLogsDisabledTransitionOnce(t *testing.T) {
	setupEmbyRefreshTestDB(t)
	buf := setEmbyRefreshTestLogger(t)
	resetEmbyRefreshScannerConfigStateForTest()

	GlobalEmbyConfig.EnableRefreshLibrary = 0
	CheckPendingEmbyLibraryRefreshTasks()
	CheckPendingEmbyLibraryRefreshTasks()
	if strings.Contains(buf.String(), "暂停待刷新任务扫描") {
		t.Fatalf("启动时未启用刷新不应写扫描暂停日志，实际日志：%s", buf.String())
	}

	GlobalEmbyConfig.EnableRefreshLibrary = 1
	CheckPendingEmbyLibraryRefreshTasks()
	GlobalEmbyConfig.EnableRefreshLibrary = 0
	CheckPendingEmbyLibraryRefreshTasks()
	CheckPendingEmbyLibraryRefreshTasks()
	if got := strings.Count(buf.String(), "暂停待刷新任务扫描"); got != 1 {
		t.Fatalf("启用变为未启用时应只写 1 条暂停日志，实际 %d，日志：%s", got, buf.String())
	}

	GlobalEmbyConfig.EnableRefreshLibrary = 1
	CheckPendingEmbyLibraryRefreshTasks()
	GlobalEmbyConfig.EnableRefreshLibrary = 0
	CheckPendingEmbyLibraryRefreshTasks()
	if got := strings.Count(buf.String(), "暂停待刷新任务扫描"); got != 2 {
		t.Fatalf("重新启用后再次关闭应再写 1 条暂停日志，实际 %d，日志：%s", got, buf.String())
	}
}

func TestRequestEmbyLibraryRefreshSkipsUnlinkedSyncPath(t *testing.T) {
	setupEmbyRefreshTestDB(t)

	if err := RequestEmbyLibraryRefreshBySyncPathId(99); err != nil {
		t.Fatalf("无媒体库关联时应安静跳过，实际错误: %v", err)
	}

	var total int64
	db.Db.Model(&EmbyLibraryRefreshTask{}).Count(&total)
	if total != 0 {
		t.Fatalf("无关联时不应创建任务，实际 %d", total)
	}
}

func TestRefreshTaskWaitsForRelatedDownloadsOnly(t *testing.T) {
	setupEmbyRefreshTestDB(t)
	db.Db.Create(&SyncFile{BaseModel: BaseModel{ID: 1}, SyncPathId: 10})
	db.Db.Create(&SyncFile{BaseModel: BaseModel{ID: 2}, SyncPathId: 20})
	db.Db.Create(&DbDownloadTask{SyncFileId: 1, Status: DownloadStatusPending})
	db.Db.Create(&DbDownloadTask{SyncFileId: 2, Status: DownloadStatusPending})

	count, err := CountActiveDownloadTasksBySyncPathIds([]uint{10})
	if err != nil {
		t.Fatalf("统计下载任务失败: %v", err)
	}
	if count != 1 {
		t.Fatalf("只应统计当前媒体库相关下载任务，实际 %d", count)
	}
}

func TestRefreshTaskWaitsForDownloadTaskWithSyncPathId(t *testing.T) {
	setupEmbyRefreshTestDB(t)
	db.Db.Create(&DbDownloadTask{SyncPathId: 10, Status: DownloadStatusPending})
	db.Db.Create(&DbDownloadTask{SyncPathId: 20, Status: DownloadStatusPending})

	count, err := CountActiveDownloadTasksBySyncPathIds([]uint{10})
	if err != nil {
		t.Fatalf("统计下载任务失败: %v", err)
	}
	if count != 1 {
		t.Fatalf("应按 sync_path_id 统计当前同步目录下载任务，实际 %d", count)
	}
}

func TestRefreshTaskKeepsSyncFileFallbackForOldDownloadTasks(t *testing.T) {
	setupEmbyRefreshTestDB(t)
	db.Db.Create(&SyncFile{BaseModel: BaseModel{ID: 1}, SyncPathId: 10})
	db.Db.Create(&DbDownloadTask{SyncFileId: 1, Status: DownloadStatusPending})

	count, err := CountActiveDownloadTasksBySyncPathIds([]uint{10})
	if err != nil {
		t.Fatalf("统计旧下载任务失败: %v", err)
	}
	if count != 1 {
		t.Fatalf("旧下载任务应继续通过 sync_file_id 兼容统计，实际 %d", count)
	}
}

func TestRefreshTaskKeepsSyncFileFallbackForNullSyncPathIDOldDownloadTasks(t *testing.T) {
	setupEmbyRefreshTestDB(t)
	db.Db.Create(&SyncFile{BaseModel: BaseModel{ID: 1}, SyncPathId: 10})
	if err := db.Db.Exec("INSERT INTO db_download_tasks (sync_file_id, sync_path_id, status) VALUES (?, NULL, ?)", 1, DownloadStatusPending).Error; err != nil {
		t.Fatalf("插入 NULL sync_path_id 旧下载任务失败: %v", err)
	}

	count, err := CountActiveDownloadTasksBySyncPathIds([]uint{10})
	if err != nil {
		t.Fatalf("统计 NULL sync_path_id 旧下载任务失败: %v", err)
	}
	if count != 1 {
		t.Fatalf("NULL sync_path_id 旧下载任务应继续通过 sync_file_id 兼容统计，实际 %d", count)
	}
}

func TestRefreshTaskWaitsForActiveSyncTask(t *testing.T) {
	setupEmbyRefreshTestDB(t)
	IsStrmSyncTaskActiveFunc = func(syncPathId uint) bool {
		return syncPathId == 10
	}

	if !HasActiveStrmSyncTask([]uint{10, 11}) {
		t.Fatal("存在活跃同步任务时应等待")
	}
	if HasActiveStrmSyncTask([]uint{11}) {
		t.Fatal("无活跃同步任务时不应等待")
	}
}

func TestRefreshTaskWaitsForSameLibraryQueuedSyncPath(t *testing.T) {
	setupEmbyRefreshTestDB(t)
	now := nowUnix()
	db.Db.Create(&EmbyLibrarySyncPath{LibraryId: "lib-movie", LibraryName: "电影", SyncPathId: 10})
	db.Db.Create(&EmbyLibrarySyncPath{LibraryId: "lib-movie", LibraryName: "电影", SyncPathId: 11})
	task := newPendingEmbyLibraryRefreshTask("lib-movie", "电影", []uint{10}, now-100)
	task.RefreshAfterAt = now - 1
	db.Db.Create(task)
	IsStrmSyncTaskActiveFunc = func(syncPathId uint) bool {
		return syncPathId == 11
	}

	ready, reason, err := IsEmbyLibraryRefreshTaskReady(task, now)
	if err != nil {
		t.Fatalf("判断刷新任务失败: %v", err)
	}
	if ready {
		t.Fatalf("同一媒体库还有同步目录在队列中时不应刷新")
	}
	if reason != "sync_running" {
		t.Fatalf("等待原因 = %s，期望 sync_running", reason)
	}
}

func TestRefreshTaskWaitsForSameLibraryDownloadOutsideTaskSyncPaths(t *testing.T) {
	setupEmbyRefreshTestDB(t)
	now := nowUnix()
	db.Db.Create(&EmbyLibrarySyncPath{LibraryId: "lib-movie", LibraryName: "电影", SyncPathId: 10})
	db.Db.Create(&EmbyLibrarySyncPath{LibraryId: "lib-movie", LibraryName: "电影", SyncPathId: 11})
	task := newPendingEmbyLibraryRefreshTask("lib-movie", "电影", []uint{10}, now-100)
	task.RefreshAfterAt = now - 1
	db.Db.Create(task)
	db.Db.Create(&DbDownloadTask{SyncPathId: 11, Status: DownloadStatusPending})

	ready, reason, err := IsEmbyLibraryRefreshTaskReady(task, now)
	if err != nil {
		t.Fatalf("判断刷新任务失败: %v", err)
	}
	if ready {
		t.Fatalf("同一媒体库其他同步目录还有下载任务时不应刷新")
	}
	if reason != "download_running" {
		t.Fatalf("等待原因 = %s，期望 download_running", reason)
	}
}

func TestRefreshTaskWaitsForRetryableFailedDownload(t *testing.T) {
	setupEmbyRefreshTestDB(t)
	db.Db.Create(&SyncFile{BaseModel: BaseModel{ID: 1}, SyncPathId: 10})
	db.Db.Create(&DbDownloadTask{SyncFileId: 1, Status: DownloadStatusFailed, RetryCount: 0})

	count, err := CountActiveDownloadTasksBySyncPathIds([]uint{10})
	if err != nil {
		t.Fatalf("统计下载任务失败: %v", err)
	}
	if count != 1 {
		t.Fatalf("可自动重试的失败任务应继续阻塞刷新，实际 %d", count)
	}
}

func TestRefreshTaskWaitsForDownloadingDownload(t *testing.T) {
	setupEmbyRefreshTestDB(t)
	db.Db.Create(&DbDownloadTask{SyncPathId: 10, Status: DownloadStatusDownloading})

	count, err := CountActiveDownloadTasksBySyncPathIds([]uint{10})
	if err != nil {
		t.Fatalf("统计下载中任务失败: %v", err)
	}
	if count != 1 {
		t.Fatalf("下载中任务应继续阻塞刷新，实际 %d", count)
	}
}

func TestScheduleNextEmbyLibraryRefreshCheckUsesEarliestFuturePendingTask(t *testing.T) {
	setupEmbyRefreshTestDB(t)
	now := nowUnix()
	dueTask := newPendingEmbyLibraryRefreshTask("lib-due", "已到期", []uint{10}, now)
	dueTask.RefreshAfterAt = now - 1
	firstFutureTask := newPendingEmbyLibraryRefreshTask("lib-first", "最早未来任务", []uint{11}, now)
	firstFutureTask.RefreshAfterAt = now + 10
	secondFutureTask := newPendingEmbyLibraryRefreshTask("lib-second", "较晚未来任务", []uint{12}, now)
	secondFutureTask.RefreshAfterAt = now + 30
	completedTask := newPendingEmbyLibraryRefreshTask("lib-completed", "已完成任务", []uint{13}, now)
	completedTask.Status = EmbyLibraryRefreshStatusCompleted
	completedTask.RefreshAfterAt = now + 5
	if err := db.Db.Create([]*EmbyLibraryRefreshTask{dueTask, firstFutureTask, secondFutureTask, completedTask}).Error; err != nil {
		t.Fatalf("创建测试刷新任务失败: %v", err)
	}

	ScheduleNextEmbyLibraryRefreshCheck()

	assertScheduledEmbyRefreshCheckAt(t, firstFutureTask.RefreshAfterAt)
}

func TestScheduleNextEmbyLibraryRefreshCheckDoesNotTriggerFuturePendingTask(t *testing.T) {
	setupEmbyRefreshTestDB(t)
	drainEmbyRefreshCheckChan()
	now := nowUnix()
	task := newPendingEmbyLibraryRefreshTask("lib-future", "未来任务", []uint{10}, now)
	task.RefreshAfterAt = now + 10
	task.LastCheckedAt = 0
	if err := db.Db.Create(task).Error; err != nil {
		t.Fatalf("创建测试刷新任务失败: %v", err)
	}

	ScheduleNextEmbyLibraryRefreshCheck()

	select {
	case <-embyRefreshCheckChan:
		t.Fatal("未到 refresh_after_at 的 pending 任务不应立即触发检查")
	default:
	}
	assertScheduledEmbyRefreshCheckAt(t, task.RefreshAfterAt)
}

func TestSetNextEmbyLibraryRefreshCheckTimerKeepsEarlierScheduleWhenLaterResultArrives(t *testing.T) {
	setupEmbyRefreshTestDB(t)
	now := nowUnix()
	earlierCheckAt := now + 10

	setNextEmbyLibraryRefreshCheckTimer(earlierCheckAt, true)
	setNextEmbyLibraryRefreshCheckTimer(now+30, true)

	assertScheduledEmbyRefreshCheckAt(t, earlierCheckAt)
}

func TestSetNextEmbyLibraryRefreshCheckTimerKeepsEarlierScheduleWhenNoTaskResultArrives(t *testing.T) {
	setupEmbyRefreshTestDB(t)
	now := nowUnix()
	earlierCheckAt := now + 10

	setNextEmbyLibraryRefreshCheckTimer(earlierCheckAt, true)
	setNextEmbyLibraryRefreshCheckTimer(0, false)

	assertScheduledEmbyRefreshCheckAt(t, earlierCheckAt)
}

func TestScheduleNextEmbyLibraryRefreshCheckIgnoresAlreadyCheckedDuePendingTasks(t *testing.T) {
	setupEmbyRefreshTestDB(t)
	now := nowUnix()
	task := newPendingEmbyLibraryRefreshTask("lib-due", "已到期", []uint{10}, now)
	task.RefreshAfterAt = now - 1
	task.LastCheckedAt = now
	if err := db.Db.Create(task).Error; err != nil {
		t.Fatalf("创建测试刷新任务失败: %v", err)
	}

	ScheduleNextEmbyLibraryRefreshCheck()

	assertNoScheduledEmbyRefreshCheck(t)
}

func TestScheduleNextEmbyLibraryRefreshCheckTriggersDueUncheckedTaskAndKeepsFutureTimer(t *testing.T) {
	setupEmbyRefreshTestDB(t)
	drainEmbyRefreshCheckChan()
	now := nowUnix()
	dueTask := newPendingEmbyLibraryRefreshTask("lib-due", "已到期未检查", []uint{10}, now)
	dueTask.RefreshAfterAt = now - 1
	dueTask.LastCheckedAt = 0
	futureTask := newPendingEmbyLibraryRefreshTask("lib-future", "未来任务", []uint{11}, now)
	futureTask.RefreshAfterAt = now + 10
	if err := db.Db.Create([]*EmbyLibraryRefreshTask{dueTask, futureTask}).Error; err != nil {
		t.Fatalf("创建测试刷新任务失败: %v", err)
	}

	ScheduleNextEmbyLibraryRefreshCheck()

	select {
	case <-embyRefreshCheckChan:
	default:
		t.Fatal("已到期且未按 refresh_after_at 检查过的 pending 任务应立即触发检查")
	}
	assertScheduledEmbyRefreshCheckAt(t, futureTask.RefreshAfterAt)
}

func TestDownloadTaskChangedSchedulesNextEmbyRefreshCheck(t *testing.T) {
	setupEmbyRefreshTestDB(t)
	now := nowUnix()
	db.Db.Create(&EmbyLibrarySyncPath{LibraryId: "lib-movie", LibraryName: "电影", SyncPathId: 10})
	task := newPendingEmbyLibraryRefreshTask("lib-movie", "电影", []uint{10}, now-100)
	task.RefreshAfterAt = now - 1
	if err := db.Db.Create(task).Error; err != nil {
		t.Fatalf("创建媒体库刷新任务失败: %v", err)
	}

	if err := NotifyEmbyRefreshDownloadTasksChangedBySyncPathIds([]uint{10}); err != nil {
		t.Fatalf("处理下载任务变化失败: %v", err)
	}

	var updated EmbyLibraryRefreshTask
	if err := db.Db.Where("library_id = ?", "lib-movie").First(&updated).Error; err != nil {
		t.Fatalf("查询媒体库刷新任务失败: %v", err)
	}
	if updated.RefreshAfterAt <= now {
		t.Fatalf("下载事件应延长稳定窗口，实际 refresh_after_at=%d now=%d", updated.RefreshAfterAt, now)
	}
	assertScheduledEmbyRefreshCheckAt(t, updated.RefreshAfterAt)
}

func TestCheckPendingEmbyLibraryRefreshTasksReschedulesFutureTask(t *testing.T) {
	setupEmbyRefreshTestDB(t)
	now := nowUnix()
	task := newPendingEmbyLibraryRefreshTask("lib-future", "未来任务", []uint{10}, now)
	task.RefreshAfterAt = now + 10
	if err := db.Db.Create(task).Error; err != nil {
		t.Fatalf("创建媒体库刷新任务失败: %v", err)
	}

	CheckPendingEmbyLibraryRefreshTasks()

	assertScheduledEmbyRefreshCheckAt(t, task.RefreshAfterAt)
}

func TestCheckPendingEmbyLibraryRefreshTasksRefreshesOnlyDueLibraries(t *testing.T) {
	setupEmbyRefreshTestDB(t)
	calledLibraries := make(chan string, 2)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		parts := strings.Split(r.URL.Path, "/")
		if len(parts) >= 5 {
			calledLibraries <- parts[3]
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()
	GlobalEmbyConfig.EmbyUrl = server.URL

	now := nowUnix()
	dueTask := newPendingEmbyLibraryRefreshTask("lib-due", "已到期媒体库", []uint{10}, now-100)
	dueTask.RefreshAfterAt = now - 1
	futureTask := newPendingEmbyLibraryRefreshTask("lib-future", "未来媒体库", []uint{11}, now)
	futureTask.RefreshAfterAt = now + 10
	if err := db.Db.Create([]*EmbyLibraryRefreshTask{dueTask, futureTask}).Error; err != nil {
		t.Fatalf("创建媒体库刷新任务失败: %v", err)
	}

	CheckPendingEmbyLibraryRefreshTasks()

	var updatedDue EmbyLibraryRefreshTask
	if err := db.Db.Where("library_id = ?", "lib-due").First(&updatedDue).Error; err != nil {
		t.Fatalf("查询已到期媒体库任务失败: %v", err)
	}
	if updatedDue.Status != EmbyLibraryRefreshStatusCompleted {
		t.Fatalf("已到期媒体库状态 = %s，期望 completed", updatedDue.Status)
	}
	var updatedFuture EmbyLibraryRefreshTask
	if err := db.Db.Where("library_id = ?", "lib-future").First(&updatedFuture).Error; err != nil {
		t.Fatalf("查询未来媒体库任务失败: %v", err)
	}
	if updatedFuture.Status != EmbyLibraryRefreshStatusPending {
		t.Fatalf("未来媒体库状态 = %s，期望 pending", updatedFuture.Status)
	}

	select {
	case got := <-calledLibraries:
		if got != "lib-due" {
			t.Fatalf("刷新媒体库 ID = %s，期望 lib-due", got)
		}
	default:
		t.Fatal("已到期媒体库应触发刷新")
	}
	select {
	case got := <-calledLibraries:
		t.Fatalf("不应刷新未到期媒体库，实际刷新 %s", got)
	default:
	}
	assertScheduledEmbyRefreshCheckAt(t, updatedFuture.RefreshAfterAt)
}

func TestEmbyRefreshTimerTriggersCheckChannel(t *testing.T) {
	setupEmbyRefreshTestDB(t)
	drainEmbyRefreshCheckChan()
	now := nowUnix()
	task := newPendingEmbyLibraryRefreshTask("lib-timer", "定时唤醒", []uint{10}, now)
	task.RefreshAfterAt = now + 1
	if err := db.Db.Create(task).Error; err != nil {
		t.Fatalf("创建媒体库刷新任务失败: %v", err)
	}

	ScheduleNextEmbyLibraryRefreshCheck()

	timeout := time.Until(time.Unix(task.RefreshAfterAt, 0)) + 500*time.Millisecond
	if timeout < 500*time.Millisecond {
		timeout = 500 * time.Millisecond
	}
	select {
	case <-embyRefreshCheckChan:
	case <-time.After(timeout):
		t.Fatalf("Emby 刷新 timer 未在 %s 内触发检查", timeout)
	}
}

func TestClearDownloadPendingTasksCancelsPendingEmbyRefreshTask(t *testing.T) {
	setupEmbyRefreshTestDB(t)
	now := nowUnix()
	db.Db.Create(&EmbyLibrarySyncPath{LibraryId: "lib-movie", LibraryName: "电影", SyncPathId: 10})
	task := newPendingEmbyLibraryRefreshTask("lib-movie", "电影", []uint{10}, now)
	if err := db.Db.Create(task).Error; err != nil {
		t.Fatalf("创建媒体库刷新任务失败: %v", err)
	}
	if err := db.Db.Create(&DbDownloadTask{
		SyncPathId: 10,
		Status:     DownloadStatusPending,
		Source:     DownloadSourceStrm,
	}).Error; err != nil {
		t.Fatalf("创建等待下载任务失败: %v", err)
	}

	if err := ClearDownloadPendingTasks(); err != nil {
		t.Fatalf("清空等待下载任务失败: %v", err)
	}

	var updated EmbyLibraryRefreshTask
	if err := db.Db.Where("library_id = ?", "lib-movie").First(&updated).Error; err != nil {
		t.Fatalf("查询媒体库刷新任务失败: %v", err)
	}
	if updated.Status != EmbyLibraryRefreshStatusCancelled {
		t.Fatalf("媒体库刷新任务状态 = %s，期望 cancelled", updated.Status)
	}
	if !strings.Contains(updated.Error, "清空等待下载任务") {
		t.Fatalf("媒体库刷新任务取消原因 = %s，期望包含 清空等待下载任务", updated.Error)
	}
}

func TestRetryFailedDownloadTasksKeepsPendingEmbyRefreshTask(t *testing.T) {
	setupEmbyRefreshTestDB(t)
	now := nowUnix()
	db.Db.Create(&EmbyLibrarySyncPath{LibraryId: "lib-movie", LibraryName: "电影", SyncPathId: 10})
	task := newPendingEmbyLibraryRefreshTask("lib-movie", "电影", []uint{10}, now)
	if err := db.Db.Create(task).Error; err != nil {
		t.Fatalf("创建媒体库刷新任务失败: %v", err)
	}
	if err := db.Db.Create(&DbDownloadTask{
		SyncPathId: 10,
		Status:     DownloadStatusFailed,
		RetryCount: 0,
		Source:     DownloadSourceStrm,
	}).Error; err != nil {
		t.Fatalf("创建失败下载任务失败: %v", err)
	}

	if err := RetryFailedDownloadTasks(DefaultQueueRetryMax); err != nil {
		t.Fatalf("重试失败下载任务失败: %v", err)
	}

	var updated EmbyLibraryRefreshTask
	if err := db.Db.Where("library_id = ?", "lib-movie").First(&updated).Error; err != nil {
		t.Fatalf("查询媒体库刷新任务失败: %v", err)
	}
	if updated.Status != EmbyLibraryRefreshStatusPending {
		t.Fatalf("媒体库刷新任务状态 = %s，期望 pending", updated.Status)
	}
}

func TestRefreshTaskCancelsAfterDeadline(t *testing.T) {
	setupEmbyRefreshTestDB(t)
	var refreshCalled atomic.Bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		refreshCalled.Store(true)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()
	GlobalEmbyConfig.EmbyUrl = server.URL

	now := nowUnix()
	task := newPendingEmbyLibraryRefreshTask("lib-movie", "电影", []uint{10}, now-DefaultEmbyRefreshMaxWaitSeconds-1)
	task.RefreshAfterAt = now - 1
	task.DeadlineAt = now - 1
	db.Db.Create(task)

	CheckPendingEmbyLibraryRefreshTasks()

	var updated EmbyLibraryRefreshTask
	db.Db.First(&updated, task.ID)
	if updated.Status != EmbyLibraryRefreshStatusCancelled {
		t.Fatalf("deadline 到期后状态 = %s，期望 cancelled", updated.Status)
	}
	if updated.LastRefreshAt != 0 {
		t.Fatalf("取消刷新后不应记录刷新时间，实际 last_refresh_at=%d", updated.LastRefreshAt)
	}
	if refreshCalled.Load() {
		t.Fatal("deadline 到期取消刷新时不应调用 Emby 刷新接口")
	}
}

func TestDownloadTaskChangedEventIsBatched(t *testing.T) {
	setupEmbyRefreshTestDB(t)
	now := nowUnix()
	db.Db.Create(&EmbyLibrarySyncPath{LibraryId: "lib-movie", LibraryName: "电影", SyncPathId: 10})
	task := newPendingEmbyLibraryRefreshTask("lib-movie", "电影", []uint{10}, now-100)
	task.RefreshAfterAt = now - 1
	db.Db.Create(task)
	db.Db.Create(&SyncFile{BaseModel: BaseModel{ID: 1}, SyncPathId: 10})

	HandleDownloadTaskStatusChanged(helpers.Event{Data: DownloadTaskStatusChangedPayload{SyncFileId: 1}})

	var updated EmbyLibraryRefreshTask
	db.Db.Where("library_id = ?", "lib-movie").First(&updated)
	if updated.RefreshAfterAt > now {
		t.Fatalf("下载事件应先进入批量队列，不应立即更新DB，实际 refresh_after_at=%d now=%d", updated.RefreshAfterAt, now)
	}

	if err := flushPendingEmbyRefreshDownloadTaskChanges(); err != nil {
		t.Fatalf("批量处理下载事件失败: %v", err)
	}

	db.Db.Where("library_id = ?", "lib-movie").First(&updated)
	if updated.RefreshAfterAt <= now {
		t.Fatalf("批量处理下载事件后应延长稳定窗口，实际 refresh_after_at=%d now=%d", updated.RefreshAfterAt, now)
	}
}

func TestDownloadTaskChangedEventUsesSyncPathId(t *testing.T) {
	setupEmbyRefreshTestDB(t)
	now := nowUnix()
	db.Db.Create(&EmbyLibrarySyncPath{LibraryId: "lib-movie", LibraryName: "电影", SyncPathId: 10})
	task := newPendingEmbyLibraryRefreshTask("lib-movie", "电影", []uint{10}, now-100)
	task.RefreshAfterAt = now - 1
	db.Db.Create(task)

	HandleDownloadTaskStatusChanged(helpers.Event{Data: DownloadTaskStatusChangedPayload{SyncPathId: 10}})

	if err := flushPendingEmbyRefreshDownloadTaskChanges(); err != nil {
		t.Fatalf("批量处理下载事件失败: %v", err)
	}

	var updated EmbyLibraryRefreshTask
	db.Db.Where("library_id = ?", "lib-movie").First(&updated)
	if updated.RefreshAfterAt <= now {
		t.Fatalf("sync_path_id 下载事件应延长稳定窗口，实际 refresh_after_at=%d now=%d", updated.RefreshAfterAt, now)
	}
}

func TestMarkEmbyRefreshTaskCompleted(t *testing.T) {
	setupEmbyRefreshTestDB(t)
	task := newPendingEmbyLibraryRefreshTask("lib-movie", "电影", []uint{10}, nowUnix())
	db.Db.Create(task)

	if err := markEmbyRefreshTaskCompleted(task); err != nil {
		t.Fatalf("标记完成失败: %v", err)
	}

	var updated EmbyLibraryRefreshTask
	db.Db.First(&updated, task.ID)
	if updated.Status != EmbyLibraryRefreshStatusCompleted {
		t.Fatalf("完成后状态 = %s，期望 completed", updated.Status)
	}
	if updated.LastRefreshAt == 0 {
		t.Fatal("完成后应记录 last_refresh_at")
	}
}
