package models

import (
	"io"
	"log"
	"testing"

	"qmediasync/internal/db"
	"qmediasync/internal/helpers"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func setupStrmGenerationTaskTestDB(t *testing.T) {
	t.Helper()
	if helpers.AppLogger == nil {
		helpers.AppLogger = &helpers.QLogger{Logger: log.New(io.Discard, "", 0)}
	}
	testDb, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("打开测试数据库失败: %v", err)
	}
	db.Db = testDb
	if err := db.Db.AutoMigrate(&StrmGenerationTask{}); err != nil {
		t.Fatalf("迁移测试表失败: %v", err)
	}
}

func TestEnqueueStrmGenerationTaskDedupesRequestHash(t *testing.T) {
	setupStrmGenerationTaskTestDB(t)

	first, err := EnqueueStrmGenerationTask(&StrmGenerationTask{
		Source:      StrmGenerationSourceWebhook,
		TaskType:    StrmGenerationTaskTypeFile,
		SyncPathId:  10,
		AccountId:   2,
		FileId:      "file-1",
		RequestHash: "sync:10:file:file-1",
	})
	if err != nil {
		t.Fatalf("首次入队失败: %v", err)
	}

	second, err := EnqueueStrmGenerationTask(&StrmGenerationTask{
		Source:      StrmGenerationSourceWebhook,
		TaskType:    StrmGenerationTaskTypeFile,
		SyncPathId:  10,
		AccountId:   2,
		FileId:      "file-1",
		RequestHash: "sync:10:file:file-1",
	})
	if err != nil {
		t.Fatalf("重复入队失败: %v", err)
	}
	if second.ID != first.ID {
		t.Fatalf("重复 request_hash 应返回已有任务，got %d want %d", second.ID, first.ID)
	}

	var count int64
	if err := db.Db.Model(&StrmGenerationTask{}).Count(&count).Error; err != nil {
		t.Fatalf("统计任务失败: %v", err)
	}
	if count != 1 {
		t.Fatalf("任务数量 = %d，期望 1", count)
	}
}

func TestEnqueueStrmGenerationFileArchivesFailedRequestHash(t *testing.T) {
	setupStrmGenerationTaskTestDB(t)

	first, err := EnqueueStrmGenerationTask(&StrmGenerationTask{
		Source:      StrmGenerationSourceWebhook,
		TaskType:    StrmGenerationTaskTypeFile,
		SyncPathId:  10,
		AccountId:   2,
		FileId:      "file-1",
		RequestHash: "sync:10:file:file-1",
	})
	if err != nil {
		t.Fatalf("首次单文件任务入队失败: %v", err)
	}
	if err := first.MarkFailed("生成 STRM 失败"); err != nil {
		t.Fatalf("标记单文件任务失败失败: %v", err)
	}

	second, err := EnqueueStrmGenerationTask(&StrmGenerationTask{
		Source:      StrmGenerationSourceWebhook,
		TaskType:    StrmGenerationTaskTypeFile,
		SyncPathId:  10,
		AccountId:   2,
		FileId:      "file-1",
		RequestHash: "sync:10:file:file-1",
	})
	if err != nil {
		t.Fatalf("失败后单文件任务重新入队失败: %v", err)
	}
	if second.ID == first.ID {
		t.Fatalf("失败任务再次入队应创建新任务，仍返回 ID %d", second.ID)
	}
	if second.Status != StrmGenerationStatusPending {
		t.Fatalf("新任务状态 = %s，期望 pending", second.Status)
	}

	var archived StrmGenerationTask
	if err := db.Db.First(&archived, first.ID).Error; err != nil {
		t.Fatalf("读取归档任务失败: %v", err)
	}
	if archived.RequestHash == "sync:10:file:file-1" {
		t.Fatalf("旧失败任务 request_hash 未归档: %s", archived.RequestHash)
	}
}

func TestEnqueueStrmGenerationDirectoryScanDedupesOnlyActiveTasks(t *testing.T) {
	setupStrmGenerationTaskTestDB(t)

	first, err := EnqueueStrmGenerationTask(&StrmGenerationTask{
		Source:        StrmGenerationSourceWebhook,
		TaskType:      StrmGenerationTaskTypeDirectoryScan,
		SyncPathId:    10,
		AccountId:     2,
		DirectoryPath: "/remote/show",
		RequestHash:   "webhook:directory:10::/remote/show",
	})
	if err != nil {
		t.Fatalf("首次目录扫描入队失败: %v", err)
	}

	activeDuplicate, err := EnqueueStrmGenerationTask(&StrmGenerationTask{
		Source:        StrmGenerationSourceWebhook,
		TaskType:      StrmGenerationTaskTypeDirectoryScan,
		SyncPathId:    10,
		AccountId:     2,
		DirectoryPath: "/remote/show",
		RequestHash:   "webhook:directory:10::/remote/show",
	})
	if err != nil {
		t.Fatalf("运行中目录扫描重复入队失败: %v", err)
	}
	if activeDuplicate.ID != first.ID {
		t.Fatalf("未完成目录扫描应去重，got %d want %d", activeDuplicate.ID, first.ID)
	}

	first.Status = StrmGenerationStatusCompleted
	if err := db.Db.Save(first).Error; err != nil {
		t.Fatalf("标记目录扫描完成失败: %v", err)
	}

	afterCompleted, err := EnqueueStrmGenerationTask(&StrmGenerationTask{
		Source:        StrmGenerationSourceWebhook,
		TaskType:      StrmGenerationTaskTypeDirectoryScan,
		SyncPathId:    10,
		AccountId:     2,
		DirectoryPath: "/remote/show",
		RequestHash:   "webhook:directory:10::/remote/show",
	})
	if err != nil {
		t.Fatalf("完成后目录扫描重新入队失败: %v", err)
	}
	if afterCompleted.ID == first.ID {
		t.Fatalf("已完成目录扫描再次请求应创建新任务，仍返回 ID %d", afterCompleted.ID)
	}

	var count int64
	if err := db.Db.Model(&StrmGenerationTask{}).Count(&count).Error; err != nil {
		t.Fatalf("统计任务失败: %v", err)
	}
	if count != 2 {
		t.Fatalf("任务数量 = %d，期望 2", count)
	}
}

func TestStrmGenerationTaskWebhookOptionsDefaultFalse(t *testing.T) {
	setupStrmGenerationTaskTestDB(t)

	task, err := EnqueueStrmGenerationTask(&StrmGenerationTask{
		Source:     StrmGenerationSourceWebhook,
		TaskType:   StrmGenerationTaskTypeFile,
		SyncPathId: 1,
		FileId:     "file-1",
		PickCode:   "pick-1",
	})
	if err != nil {
		t.Fatalf("创建 STRM 任务失败: %v", err)
	}
	if task.DownloadMeta || task.RefreshEmby {
		t.Fatalf("Webhook 开关默认值 = download_meta:%v refresh_emby:%v，期望均为 false", task.DownloadMeta, task.RefreshEmby)
	}
}

func TestStrmGenerationTaskRefreshTargetsMergeAndLibraryOverride(t *testing.T) {
	task := &StrmGenerationTask{}
	task.MergeRefreshTargets([]EmbyRefreshTarget{
		{
			TargetType:        EmbyRefreshTargetTypeItem,
			ItemID:            "movie-1",
			ItemName:          "电影 1",
			FallbackLibraryId: "lib-movie",
		},
		{
			TargetType:          EmbyRefreshTargetTypeLibrary,
			FallbackLibraryId:   "lib-movie",
			FallbackLibraryName: "电影库",
		},
		{
			TargetType:        EmbyRefreshTargetTypeItem,
			ItemID:            "movie-2",
			ItemName:          "电影 2",
			FallbackLibraryId: "lib-movie",
		},
		{
			TargetType:        EmbyRefreshTargetTypeItem,
			ItemID:            "season-1",
			ItemName:          "第一季",
			ItemType:          "Season",
			Recursive:         true,
			FallbackLibraryId: "lib-tv",
		},
	})

	got := task.GetRefreshTargets()
	if len(got) != 2 {
		t.Fatalf("刷新目标数量 = %d，期望电影库 + season 两个目标: %+v", len(got), got)
	}
	if got[0].TargetType != EmbyRefreshTargetTypeLibrary || got[0].FallbackLibraryId != "lib-movie" {
		t.Fatalf("第一个目标 = %+v，期望 lib-movie 媒体库目标", got[0])
	}
	if got[1].TargetType != EmbyRefreshTargetTypeItem || got[1].ItemID != "season-1" {
		t.Fatalf("第二个目标 = %+v，期望 season-1 item 目标", got[1])
	}
}

func TestStrmGenerationParentReadyForRefresh(t *testing.T) {
	parent := &StrmGenerationTask{
		TotalItems:    3,
		AcceptedItems: 2,
		FailedItems:   1,
		ChangedItems:  1,
		RefreshEmby:   true,
	}
	if !parent.IsReadyToSubmitRefresh() {
		t.Fatal("全部子任务完成且有变化时应可提交刷新")
	}
	parent.RefreshSubmitted = true
	if parent.IsReadyToSubmitRefresh() {
		t.Fatal("已提交刷新后不应再次提交")
	}
}

func TestStrmGenerationTaskRetryAndRunningReset(t *testing.T) {
	setupStrmGenerationTaskTestDB(t)

	task, err := EnqueueStrmGenerationTask(&StrmGenerationTask{
		Source:       StrmGenerationSourceUploadCompleted,
		TaskType:     StrmGenerationTaskTypeFile,
		SyncPathId:   10,
		AccountId:    2,
		FileId:       "file-2",
		UploadTaskId: 99,
		RequestHash:  "sync:10:file:file-2",
	})
	if err != nil {
		t.Fatalf("入队失败: %v", err)
	}
	if task.UploadTaskId != 99 {
		t.Fatalf("upload_task_id = %d，期望 99", task.UploadTaskId)
	}

	if err := task.MarkFailed("生成 STRM 失败"); err != nil {
		t.Fatalf("标记失败失败: %v", err)
	}
	var failed StrmGenerationTask
	if err := db.Db.First(&failed, task.ID).Error; err != nil {
		t.Fatalf("读取失败任务失败: %v", err)
	}
	if failed.Status != StrmGenerationStatusFailed || failed.RetryCount != 1 || failed.LastError != "生成 STRM 失败" {
		t.Fatalf("失败任务 = %+v，期望 failed/retry_count=1/last_error", failed)
	}

	running := &StrmGenerationTask{
		Source:     StrmGenerationSourceWebhook,
		TaskType:   StrmGenerationTaskTypeFile,
		SyncPathId: 10,
		AccountId:  2,
		FileId:     "file-running",
		Status:     StrmGenerationStatusRunning,
	}
	if err := db.Db.Create(running).Error; err != nil {
		t.Fatalf("创建 running 任务失败: %v", err)
	}
	if err := ResetRunningStrmGenerationTasks(); err != nil {
		t.Fatalf("恢复 running 任务失败: %v", err)
	}
	var reset StrmGenerationTask
	if err := db.Db.First(&reset, running.ID).Error; err != nil {
		t.Fatalf("读取重置任务失败: %v", err)
	}
	if reset.Status != StrmGenerationStatusPending {
		t.Fatalf("running 重置后 status = %s，期望 pending", reset.Status)
	}
}

func TestStrmGenerationTaskRunningAndCompletedTransitions(t *testing.T) {
	setupStrmGenerationTaskTestDB(t)

	task, err := EnqueueStrmGenerationTask(&StrmGenerationTask{
		Source:     StrmGenerationSourceUploadCompleted,
		TaskType:   StrmGenerationTaskTypeFile,
		SyncPathId: 10,
		AccountId:  2,
		FileId:     "file-transition",
	})
	if err != nil {
		t.Fatalf("入队失败: %v", err)
	}

	if err := task.MarkRunning(); err != nil {
		t.Fatalf("标记 running 失败: %v", err)
	}
	var running StrmGenerationTask
	if err := db.Db.First(&running, task.ID).Error; err != nil {
		t.Fatalf("读取 running 任务失败: %v", err)
	}
	if running.Status != StrmGenerationStatusRunning || running.LastError != "" {
		t.Fatalf("running 任务 = %+v，期望 status=running 且清空 last_error", running)
	}

	if err := running.MarkCompleted(); err != nil {
		t.Fatalf("标记 completed 失败: %v", err)
	}
	var completed StrmGenerationTask
	if err := db.Db.First(&completed, task.ID).Error; err != nil {
		t.Fatalf("读取 completed 任务失败: %v", err)
	}
	if completed.Status != StrmGenerationStatusCompleted || completed.LastError != "" {
		t.Fatalf("completed 任务 = %+v，期望 completed 且无错误", completed)
	}
}

func TestGetPendingStrmGenerationTasksFiltersAndOrders(t *testing.T) {
	setupStrmGenerationTaskTestDB(t)

	first, err := EnqueueStrmGenerationTask(&StrmGenerationTask{
		Source:     StrmGenerationSourceWebhook,
		TaskType:   StrmGenerationTaskTypeFile,
		SyncPathId: 10,
		AccountId:  2,
		FileId:     "file-pending-1",
	})
	if err != nil {
		t.Fatalf("创建第一个 pending 任务失败: %v", err)
	}
	if _, err := EnqueueStrmGenerationTask(&StrmGenerationTask{
		Source:     StrmGenerationSourceWebhook,
		TaskType:   StrmGenerationTaskTypeFile,
		SyncPathId: 10,
		AccountId:  2,
		FileId:     "file-running",
		Status:     StrmGenerationStatusRunning,
	}); err != nil {
		t.Fatalf("创建 running 任务失败: %v", err)
	}
	if _, err := EnqueueStrmGenerationTask(&StrmGenerationTask{
		Source:     StrmGenerationSourceWebhook,
		TaskType:   StrmGenerationTaskTypeFile,
		SyncPathId: 10,
		AccountId:  2,
		FileId:     "file-failed",
		Status:     StrmGenerationStatusFailed,
	}); err != nil {
		t.Fatalf("创建 failed 任务失败: %v", err)
	}
	second, err := EnqueueStrmGenerationTask(&StrmGenerationTask{
		Source:     StrmGenerationSourceWebhook,
		TaskType:   StrmGenerationTaskTypeFile,
		SyncPathId: 10,
		AccountId:  2,
		FileId:     "file-pending-2",
	})
	if err != nil {
		t.Fatalf("创建第二个 pending 任务失败: %v", err)
	}

	tasks, err := GetPendingStrmGenerationTasks(1)
	if err != nil {
		t.Fatalf("查询 pending 任务失败: %v", err)
	}
	if len(tasks) != 1 || tasks[0].ID != first.ID {
		t.Fatalf("limit=1 返回 = %+v，期望只返回第一个 pending", tasks)
	}

	tasks, err = GetPendingStrmGenerationTasks(10)
	if err != nil {
		t.Fatalf("查询 pending 任务失败: %v", err)
	}
	if len(tasks) != 2 || tasks[0].ID != first.ID || tasks[1].ID != second.ID {
		t.Fatalf("pending 任务 = %+v，期望按 ID 返回两个 pending", tasks)
	}
}

func TestStrmGenerationDirectoryParentChildStats(t *testing.T) {
	setupStrmGenerationTaskTestDB(t)

	parent, err := EnqueueStrmGenerationTask(&StrmGenerationTask{
		Source:        StrmGenerationSourceWebhook,
		TaskType:      StrmGenerationTaskTypeDirectoryScan,
		SyncPathId:    10,
		AccountId:     2,
		DirectoryId:   "dir-1",
		DirectoryPath: "/remote/show",
		RequestHash:   "sync:10:dir:dir-1",
	})
	if err != nil {
		t.Fatalf("目录扫描父任务入队失败: %v", err)
	}

	child, err := EnqueueStrmGenerationTask(&StrmGenerationTask{
		Source:       StrmGenerationSourceWebhook,
		TaskType:     StrmGenerationTaskTypeFile,
		ParentTaskId: parent.ID,
		SyncPathId:   10,
		AccountId:    2,
		FileId:       "file-child",
		RequestHash:  "sync:10:file:file-child",
	})
	if err != nil {
		t.Fatalf("目录扫描子任务入队失败: %v", err)
	}
	if child.ParentTaskId != parent.ID {
		t.Fatalf("parent_task_id = %d，期望 %d", child.ParentTaskId, parent.ID)
	}

	if err := IncrementStrmGenerationDirectoryStats(parent.ID, 1, 2); err != nil {
		t.Fatalf("累计目录扫描统计失败: %v", err)
	}
	var got StrmGenerationTask
	if err := db.Db.First(&got, parent.ID).Error; err != nil {
		t.Fatalf("读取目录扫描父任务失败: %v", err)
	}
	if got.AcceptedItems != 1 || got.FailedItems != 2 || got.TotalItems != 3 {
		t.Fatalf("目录统计 = accepted:%d failed:%d total:%d，期望 1/2/3", got.AcceptedItems, got.FailedItems, got.TotalItems)
	}
}

func TestStrmGenerationDirectoryStatsKeepsExpandedTotal(t *testing.T) {
	setupStrmGenerationTaskTestDB(t)

	parent, err := EnqueueStrmGenerationTask(&StrmGenerationTask{
		Source:        StrmGenerationSourceWebhook,
		TaskType:      StrmGenerationTaskTypeDirectoryScan,
		SyncPathId:    10,
		AccountId:     2,
		DirectoryId:   "dir-1",
		DirectoryPath: "/remote/show",
		TotalItems:    2,
		RequestHash:   "sync:10:dir:dir-1",
	})
	if err != nil {
		t.Fatalf("目录扫描父任务入队失败: %v", err)
	}

	if err := IncrementStrmGenerationDirectoryStats(parent.ID, 1, 0); err != nil {
		t.Fatalf("累计目录扫描统计失败: %v", err)
	}
	var got StrmGenerationTask
	if err := db.Db.First(&got, parent.ID).Error; err != nil {
		t.Fatalf("读取目录扫描父任务失败: %v", err)
	}
	if got.AcceptedItems != 1 || got.FailedItems != 0 || got.TotalItems != 2 {
		t.Fatalf("目录统计 = accepted:%d failed:%d total:%d，期望 1/0/2", got.AcceptedItems, got.FailedItems, got.TotalItems)
	}
}
