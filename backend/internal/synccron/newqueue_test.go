package synccron

import (
	"Q115-STRM/internal/models"
	"testing"
	"time"
)

func TestNewSyncQueuePerType(t *testing.T) {
	queue := NewQueuePerType(models.SourceType115)
	if queue == nil {
		t.Fatal("Failed to create queue")
	}
	if queue.sourceType != models.SourceType115 {
		t.Errorf("Expected source type %s, got %s", models.SourceType115, queue.sourceType)
	}
	if queue.status != QueueStatusRunning {
		t.Errorf("Expected status %s, got %s", QueueStatusRunning, queue.status)
	}
}

func TestAddTask(t *testing.T) {
	queue := NewQueuePerType(models.SourceType115)

	task := &NewSyncTask{
		ID:       1,
		TaskType: SyncTaskTypeStrm,
	}

	err := queue.AddTask(task)
	if err != nil {
		t.Fatalf("Failed to add task: %v", err)
	}

	if !queue.isTaskExists(task) {
		t.Error("Task should exist after adding")
	}

	duplicateTask := &NewSyncTask{
		ID:       1,
		TaskType: SyncTaskTypeStrm,
	}

	err = queue.AddTask(duplicateTask)
	if err == nil {
		t.Error("Should return error when adding duplicate task")
	}
}

func TestTaskStatus(t *testing.T) {
	queue := NewQueuePerType(models.SourceType115)

	status := queue.CheckTaskStatus(1, SyncTaskTypeStrm)
	if status != TaskStatusNone {
		t.Errorf("Expected status %d, got %d", TaskStatusNone, status)
	}

	task := &NewSyncTask{
		ID:       1,
		TaskType: SyncTaskTypeStrm,
	}

	err := queue.AddTask(task)
	if err != nil {
		t.Fatalf("Failed to add task: %v", err)
	}

	status = queue.CheckTaskStatus(1, SyncTaskTypeStrm)
	if status != TaskStatusWaiting {
		t.Errorf("Expected status %d, got %d", TaskStatusWaiting, status)
	}
}

func TestCancelTask(t *testing.T) {
	queue := NewQueuePerType(models.SourceType115)

	task := &NewSyncTask{
		ID:       1,
		TaskType: SyncTaskTypeStrm,
	}

	err := queue.AddTask(task)
	if err != nil {
		t.Fatalf("Failed to add task: %v", err)
	}

	status := queue.CheckTaskStatus(1, SyncTaskTypeStrm)
	if status != TaskStatusWaiting {
		t.Errorf("Expected status %d, got %d", TaskStatusWaiting, status)
	}

	err = queue.CancelTask(1, SyncTaskTypeStrm)
	if err != nil {
		t.Fatalf("Failed to cancel task: %v", err)
	}

	status = queue.CheckTaskStatus(1, SyncTaskTypeStrm)
	if status != TaskStatusNone {
		t.Errorf("Expected status %d after cancellation, got %d", TaskStatusNone, status)
	}
}

func TestPauseResume(t *testing.T) {
	queue := NewQueuePerType(models.SourceType115)

	task := &NewSyncTask{
		ID:       1,
		TaskType: SyncTaskTypeStrm,
	}

	err := queue.AddTask(task)
	if err != nil {
		t.Fatalf("Failed to add task: %v", err)
	}

	queue.Pause()
	statusInfo := queue.GetStatus()
	if statusInfo["status"] != QueueStatusPaused {
		t.Errorf("Expected status %s, got %s", QueueStatusPaused, statusInfo["status"])
	}

	task2 := &NewSyncTask{
		ID:       2,
		TaskType: SyncTaskTypeStrm,
	}

	err = queue.AddTask(task2)
	if err != nil {
		t.Fatalf("Failed to add task during pause: %v", err)
	}

	queue.Resume()
	statusInfo = queue.GetStatus()
	if statusInfo["status"] != QueueStatusRunning {
		t.Errorf("Expected status %s, got %s", QueueStatusRunning, statusInfo["status"])
	}

	time.Sleep(100 * time.Millisecond)
}

func TestNewSyncQueueManager(t *testing.T) {
	manager := InitNewSyncQueueManager()
	if manager == nil {
		t.Fatal("Failed to create manager")
	}

	task := &NewSyncTask{
		ID:       1,
		TaskType: SyncTaskTypeStrm,
	}

	queue := manager.getQueue(models.SourceType115)
	err := queue.AddTask(task)
	if err != nil {
		t.Fatalf("Failed to add task: %v", err)
	}

	status := queue.CheckTaskStatus(1, SyncTaskTypeStrm)
	if status != TaskStatusWaiting {
		t.Errorf("Expected status %d, got %d", TaskStatusWaiting, status)
	}
}

func TestPauseResumeAll(t *testing.T) {
	manager := InitNewSyncQueueManager()

	task1 := &NewSyncTask{
		ID:       1,
		TaskType: SyncTaskTypeStrm,
	}
	task2 := &NewSyncTask{
		ID:       2,
		TaskType: SyncTaskTypeStrm,
	}

	queue115 := manager.getQueue(models.SourceType115)
	queueLocal := manager.getQueue(models.SourceTypeLocal)

	queue115.AddTask(task1)
	queueLocal.AddTask(task2)

	manager.PauseAll()

	status115 := queue115.GetStatus()
	statusLocal := queueLocal.GetStatus()

	if status115["status"] != QueueStatusPaused {
		t.Errorf("Expected 115 queue status %s, got %s", QueueStatusPaused, status115["status"])
	}
	if statusLocal["status"] != QueueStatusPaused {
		t.Errorf("Expected Local queue status %s, got %s", QueueStatusPaused, statusLocal["status"])
	}

	manager.ResumeAll()

	status115 = queue115.GetStatus()
	statusLocal = queueLocal.GetStatus()

	if status115["status"] != QueueStatusRunning {
		t.Errorf("Expected 115 queue status %s, got %s", QueueStatusRunning, status115["status"])
	}
	if statusLocal["status"] != QueueStatusRunning {
		t.Errorf("Expected Local queue status %s, got %s", QueueStatusRunning, statusLocal["status"])
	}

	time.Sleep(100 * time.Millisecond)
}
