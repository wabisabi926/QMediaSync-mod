package controllers

import (
	"context"
	"net/http"
	"time"

	"qmediasync/internal/logstream"
	"qmediasync/internal/models"
	"qmediasync/internal/requests"
	ws "qmediasync/internal/websocket"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

const (
	syncTaskStreamVersion        = 1
	syncTaskStreamSnapshot       = "snapshot"
	syncTaskStreamTaskPatch      = "task_patch"
	syncTaskStreamLogAppend      = "log_append"
	syncTaskStreamComplete       = "complete"
	syncTaskStreamHeartbeat      = "heartbeat"
	syncTaskStreamError          = "error"
	syncTaskStreamResyncRequired = "resync_required"
)

type syncTaskStreamMessage struct {
	Type       string `json:"type"`
	Version    int    `json:"version"`
	SyncID     uint   `json:"sync_id"`
	Sequence   uint64 `json:"sequence,omitempty"`
	ServerTime int64  `json:"server_time"`
	Data       any    `json:"data,omitempty"`
}

type syncTaskSnapshot struct {
	Task      *models.Sync      `json:"task"`
	Logs      []logstream.Entry `json:"logs"`
	LogCursor int64             `json:"log_cursor"`
	LogPath   string            `json:"log_path"`
}

func buildSyncTaskSnapshotMessage(task *models.Sync, logs []logstream.Entry, logCursor int64, sequence uint64) syncTaskStreamMessage {
	return syncTaskStreamMessage{
		Type:       syncTaskStreamSnapshot,
		Version:    syncTaskStreamVersion,
		SyncID:     task.ID,
		Sequence:   sequence,
		ServerTime: time.Now().Unix(),
		Data: syncTaskSnapshot{
			Task:      task,
			Logs:      logs,
			LogCursor: logCursor,
			LogPath:   models.SyncLogRelativePath(task.ID),
		},
	}
}

var syncTaskStreamUpgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// SyncTaskStream 推送同步任务详情快照、状态 patch 和日志增量。
func SyncTaskStream(c *gin.Context) {
	idReq, err := requests.ParsePositiveIDRequest(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, APIResponse[any]{Code: BadRequest, Message: "ID 参数格式错误", Data: nil})
		return
	}

	task, err := models.GetSyncByID(idReq.ID)
	if err != nil || task == nil {
		c.JSON(http.StatusNotFound, APIResponse[any]{Code: BadRequest, Message: "同步任务不存在", Data: nil})
		return
	}

	conn, err := syncTaskStreamUpgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}
	defer conn.Close()

	ctx, cancel := context.WithCancel(c.Request.Context())
	defer cancel()

	go func() {
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				cancel()
				return
			}
		}
	}()

	taskEvents, unsubscribeTask := ws.GlobalSyncTaskHub.Subscribe(task.ID, 128)
	defer unsubscribeTask()

	latestTask, err := models.GetSyncByID(idReq.ID)
	if err != nil || latestTask == nil {
		payload := task.SyncTaskEventPayload()
		payload.Deleted = true
		_ = writeSyncTaskStreamMessage(conn, syncTaskStreamMessage{
			Type:       syncTaskStreamComplete,
			Version:    syncTaskStreamVersion,
			SyncID:     payload.SyncID,
			ServerTime: time.Now().Unix(),
			Data:       payload,
		})
		return
	}
	task = latestTask

	fullLogPath := models.SyncLogFullPath(task.ID)
	logs, cursor, err := logstream.ReadTailEntries(fullLogPath, 1000)
	if err != nil {
		_ = writeSyncTaskStreamMessage(conn, syncTaskStreamMessage{
			Type:       syncTaskStreamError,
			Version:    syncTaskStreamVersion,
			SyncID:     task.ID,
			ServerTime: time.Now().Unix(),
			Data:       map[string]string{"message": "读取同步日志失败"},
		})
		return
	}

	logEvents, unsubscribeLog, err := logstream.GlobalManager.Subscribe(ctx, fullLogPath, cursor, 512)
	if err != nil {
		_ = writeSyncTaskStreamMessage(conn, syncTaskStreamMessage{
			Type:       syncTaskStreamError,
			Version:    syncTaskStreamVersion,
			SyncID:     task.ID,
			ServerTime: time.Now().Unix(),
			Data:       map[string]string{"message": err.Error()},
		})
		return
	}
	defer unsubscribeLog()

	if err := writeSyncTaskStreamMessage(conn, buildSyncTaskSnapshotMessage(task, logs, cursor, 0)); err != nil {
		return
	}

	if task.Status == models.SyncStatusCompleted || task.Status == models.SyncStatusFailed {
		_ = writeSyncTaskStreamMessage(conn, syncTaskStreamMessage{
			Type:       syncTaskStreamComplete,
			Version:    syncTaskStreamVersion,
			SyncID:     task.ID,
			ServerTime: time.Now().Unix(),
			Data:       task.SyncTaskEventPayload(),
		})
		return
	}

	heartbeat := time.NewTicker(15 * time.Second)
	defer heartbeat.Stop()

	var completeC <-chan time.Time
	var terminalPayload ws.SyncTaskEventPayload
	for {
		select {
		case <-ctx.Done():
			return
		case <-heartbeat.C:
			if err := writeSyncTaskStreamMessage(conn, syncTaskStreamMessage{
				Type:       syncTaskStreamHeartbeat,
				Version:    syncTaskStreamVersion,
				SyncID:     task.ID,
				ServerTime: time.Now().Unix(),
			}); err != nil {
				return
			}
		case payload, ok := <-taskEvents:
			if !ok {
				return
			}
			msgType := syncTaskStreamTaskPatch
			if payload.ResyncReason != "" {
				msgType = syncTaskStreamResyncRequired
			}
			if payload.Deleted {
				_ = writeSyncTaskStreamMessage(conn, syncTaskStreamMessage{
					Type:       syncTaskStreamComplete,
					Version:    syncTaskStreamVersion,
					SyncID:     payload.SyncID,
					Sequence:   payload.Sequence,
					ServerTime: time.Now().Unix(),
					Data:       payload,
				})
				return
			}
			if err := writeSyncTaskStreamMessage(conn, syncTaskStreamMessage{
				Type:       msgType,
				Version:    syncTaskStreamVersion,
				SyncID:     payload.SyncID,
				Sequence:   payload.Sequence,
				ServerTime: time.Now().Unix(),
				Data:       payload,
			}); err != nil {
				return
			}
			if payload.Status == int(models.SyncStatusCompleted) || payload.Status == int(models.SyncStatusFailed) {
				terminalPayload = payload
				completeC = time.After(2 * time.Second)
			}
		case logMsg, ok := <-logEvents:
			if !ok {
				return
			}
			msgType := syncTaskStreamLogAppend
			data := any(logMsg)
			if logMsg.Type == "resync_required" {
				msgType = syncTaskStreamResyncRequired
			}
			if logMsg.Type == "error" {
				msgType = syncTaskStreamError
			}
			if err := writeSyncTaskStreamMessage(conn, syncTaskStreamMessage{
				Type:       msgType,
				Version:    syncTaskStreamVersion,
				SyncID:     task.ID,
				ServerTime: time.Now().Unix(),
				Data:       data,
			}); err != nil {
				return
			}
		case <-completeC:
			_ = writeSyncTaskStreamMessage(conn, syncTaskStreamMessage{
				Type:       syncTaskStreamComplete,
				Version:    syncTaskStreamVersion,
				SyncID:     terminalPayload.SyncID,
				Sequence:   terminalPayload.Sequence,
				ServerTime: time.Now().Unix(),
				Data:       terminalPayload,
			})
			return
		}
	}
}

func writeSyncTaskStreamMessage(conn *websocket.Conn, msg syncTaskStreamMessage) error {
	if msg.ServerTime == 0 {
		msg.ServerTime = time.Now().Unix()
	}
	return conn.WriteJSON(msg)
}
