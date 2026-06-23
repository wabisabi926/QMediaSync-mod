package models

import "testing"

func TestCanRetryFailedDownloadTask(t *testing.T) {
	task := &DbDownloadTask{Status: DownloadStatusFailed, Error: "network", RetryCount: 0}
	task.PrepareDownloadRetry(1)
	if task.Status != DownloadStatusPending {
		t.Fatal("下载失败任务应回到等待中")
	}
	if task.Error != "" {
		t.Fatal("重试时应清空错误信息")
	}
	if task.RetryCount != 1 {
		t.Fatal("重试次数应递增")
	}
}

func TestDownloadTaskDoesNotRetryBeyondLimit(t *testing.T) {
	task := &DbDownloadTask{Status: DownloadStatusFailed, RetryCount: 1}
	if task.CanRetry(1) {
		t.Fatal("达到最大重试次数后不应继续重试")
	}
}

func TestCanRetryFailedUploadTask(t *testing.T) {
	task := &DbUploadTask{Status: UploadStatusFailed, Error: "network", RetryCount: 0}
	task.PrepareUploadRetry(1)
	if task.Status != UploadStatusPending {
		t.Fatal("上传失败任务应回到等待中")
	}
	if task.Error != "" {
		t.Fatal("重试时应清空错误信息")
	}
	if task.RetryCount != 1 {
		t.Fatal("重试次数应递增")
	}
}

func TestUploadTaskDoesNotRetryBeyondLimit(t *testing.T) {
	task := &DbUploadTask{Status: UploadStatusFailed, RetryCount: 1}
	if task.CanRetry(1) {
		t.Fatal("达到最大重试次数后不应继续重试")
	}
}
