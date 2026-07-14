package models

import (
	"qmediasync/internal/db"
	"qmediasync/internal/helpers"
)

// QueueStatusSnapshot 队列运行状态和任务状态统计快照。
type QueueStatusSnapshot struct {
	Running    bool  `json:"running"`
	Pending    int64 `json:"pending"`
	Processing int64 `json:"processing"`
	Completed  int64 `json:"completed"`
	Failed     int64 `json:"failed"`
	Cancelled  int64 `json:"cancelled"`
	Total      int64 `json:"total"`
}

type queueStatusCount struct {
	Status int
	Count  int64
}

func buildQueueStatusSnapshot(running bool, rows []queueStatusCount) QueueStatusSnapshot {
	snapshot := QueueStatusSnapshot{Running: running}
	for _, row := range rows {
		snapshot.Total += row.Count
		switch row.Status {
		case int(UploadStatusPending):
			snapshot.Pending += row.Count
		case int(UploadStatusUploading), int(UploadStatusRemoteCompletedPendingFinalize), int(UploadStatusRemoteCompletedFinalizing):
			snapshot.Processing += row.Count
		case int(UploadStatusCompleted):
			snapshot.Completed += row.Count
		case int(UploadStatusFailed):
			snapshot.Failed += row.Count
		case int(UploadStatusCancelled):
			snapshot.Cancelled += row.Count
		}
	}
	return snapshot
}

func queryQueueStatusSnapshot(model any, running bool) QueueStatusSnapshot {
	var rows []queueStatusCount
	if err := db.Db.Model(model).
		Select("status, count(*) as count").
		Group("status").
		Scan(&rows).Error; err != nil {
		helpers.AppLogger.Errorf("查询队列状态快照失败：%v", err)
		return QueueStatusSnapshot{Running: running}
	}
	return buildQueueStatusSnapshot(running, rows)
}

// GetDownloadQueueStatusSnapshot 获取下载队列状态快照。
func GetDownloadQueueStatusSnapshot() QueueStatusSnapshot {
	running := GlobalDownloadQueue != nil && GlobalDownloadQueue.IsRunning()
	return queryQueueStatusSnapshot(&DbDownloadTask{}, running)
}

// GetUploadQueueStatusSnapshot 获取上传队列状态快照。
func GetUploadQueueStatusSnapshot() QueueStatusSnapshot {
	running := GlobalUploadQueue != nil && GlobalUploadQueue.IsRunning()
	return queryQueueStatusSnapshot(&DbUploadTask{}, running)
}
