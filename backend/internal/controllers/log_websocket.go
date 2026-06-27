package controllers

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"qmediasync/internal/helpers"
	"qmediasync/internal/logstream"
	"qmediasync/internal/requests"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// LogEntry 日志条目结构。
type LogEntry = logstream.Entry

// parseLogLine 解析日志行，提取级别、消息和时间戳。
func parseLogLine(line string) LogEntry {
	return logstream.ParseLine(line)
}

type OldLogsResponse struct {
	Entries  []LogEntry `json:"entries"`
	Pos      int64      `json:"pos"`
	StartPos int64      `json:"start_pos"`
}

// GetOldLogs 通过 HTTP 接口获取旧日志，返回 JSON 格式。
func GetOldLogs(c *gin.Context) {
	var req requests.OldLogsRequest
	if err := c.ShouldBind(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求参数"})
		return
	}
	if err := req.Validate(); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	pos := req.Pos
	limit := req.Limit
	direction := req.Direction

	if pos == 0 && direction == "forward" {
		// 已经到了文件开头
		// 返回 JSON 结果
		c.JSON(http.StatusOK, OldLogsResponse{
			Entries:  make([]LogEntry, 0),
			Pos:      0,
			StartPos: 0,
		})
		return
	}

	// 拼接完整日志文件路径
	fullLogPath, err := helpers.SafeJoin(filepath.Join(helpers.ConfigDir, "logs"), req.Path)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("日志文件路径不合法：%v", err)})
		return
	}

	// 检查文件是否存在
	if _, serr := os.Stat(fullLogPath); os.IsNotExist(serr) {
		c.JSON(http.StatusNotFound, gin.H{"error": "日志文件不存在"})
		return
	}
	resultLines, newPos, err := helpers.ReadLines(fullLogPath, int64(pos), limit, direction)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("读取日志文件失败：%v", err)})
		return
	}
	// 解析日志行，转换为 LogEntry 结构
	entries := make([]LogEntry, 0, len(resultLines))
	for _, line := range resultLines {
		entries = append(entries, parseLogLine(line))
	}

	// 返回 JSON 结果
	c.JSON(http.StatusOK, OldLogsResponse{
		Entries:  entries,
		Pos:      newPos,
		StartPos: int64(pos),
	})
}

// DownloadLogFile 下载日志文件
func DownloadLogFile(c *gin.Context) {
	var req requests.LogFileRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求参数"})
		return
	}
	if err := req.Validate(); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 拼接完整日志文件路径
	fullLogPath, err := helpers.SafeJoin(filepath.Join(helpers.ConfigDir, "logs"), req.Path)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("日志文件路径不合法：%v", err)})
		return
	}

	// 检查文件是否存在
	if _, err := os.Stat(fullLogPath); os.IsNotExist(err) {
		c.JSON(http.StatusNotFound, gin.H{"error": "日志文件不存在"})
		return
	}

	// 设置响应头，支持文件下载
	c.Header("Content-Description", "File Transfer")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filepath.Base(fullLogPath)))
	c.Header("Content-Type", "application/octet-stream")
	c.Header("Content-Transfer-Encoding", "binary")
	c.Header("Expires", "0")
	c.Header("Cache-Control", "must-revalidate")
	c.Header("Pragma", "public")

	// 发送文件
	c.File(fullLogPath)
}

// LogWebSocket 通过 WebSocket 查看日志。
func LogWebSocket(c *gin.Context) {
	// 升级 HTTP 连接为 WebSocket 连接
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		helpers.AppLogger.Errorf("升级 WebSocket 连接失败：%v", err)
		return
	}
	defer conn.Close()

	req := requests.LogFileRequest{Path: c.Query("path")}
	if err := req.Validate(); err != nil {
		entry := LogEntry{
			Level:     "error",
			Message:   "错误：" + err.Error(),
			Timestamp: time.Now().Format("2006-01-02 15:04:05.000000"),
		}
		if werr := conn.WriteJSON(entry); werr != nil {
			helpers.AppLogger.Errorf("发送错误消息失败：%v", werr)
		}
		return
	}

	// 拼接完整日志文件路径
	fullLogPath, err := helpers.SafeJoin(filepath.Join(helpers.ConfigDir, "logs"), req.Path)
	if err != nil {
		entry := LogEntry{
			Level:     "error",
			Message:   fmt.Sprintf("错误：日志文件路径不合法：%v", err),
			Timestamp: time.Now().Format("2006-01-02 15:04:05.000000"),
		}
		if werr := conn.WriteJSON(entry); werr != nil {
			helpers.AppLogger.Errorf("发送错误消息失败：%v", werr)
		}
		return
	}

	// 检查文件是否存在
	if _, serr := os.Stat(fullLogPath); os.IsNotExist(serr) {
		entry := LogEntry{
			Level:     "error",
			Message:   fmt.Sprintf("错误：日志文件不存在：%s", fullLogPath),
			Timestamp: time.Now().Format("2006-01-02 15:04:05.000000"),
		}
		if werr := conn.WriteJSON(entry); werr != nil {
			helpers.AppLogger.Errorf("发送错误消息失败：%v", werr)
		}
		return
	}

	cursor, err := logstream.ReadEndCursor(fullLogPath)
	if err != nil {
		entry := LogEntry{
			Level:     "error",
			Message:   fmt.Sprintf("错误：读取日志位置失败：%v", err),
			Timestamp: time.Now().Format("2006-01-02 15:04:05.000000"),
		}
		if werr := conn.WriteJSON(entry); werr != nil {
			helpers.AppLogger.Errorf("发送错误消息失败：%v", werr)
		}
		return
	}

	ctx, cancel := context.WithCancel(c.Request.Context())
	defer cancel()

	messages, unsubscribe, err := logstream.GlobalManager.Subscribe(ctx, fullLogPath, cursor, 256)
	if err != nil {
		entry := LogEntry{
			Level:     "error",
			Message:   fmt.Sprintf("错误：订阅日志文件失败：%v", err),
			Timestamp: time.Now().Format("2006-01-02 15:04:05.000000"),
		}
		if werr := conn.WriteJSON(entry); werr != nil {
			helpers.AppLogger.Errorf("发送错误消息失败：%v", werr)
		}
		return
	}
	defer unsubscribe()

	// 启动协程处理客户端消息
	go func() {
		for {
			_, _, err := conn.ReadMessage()
			if err != nil {
				cancel()
				return
			}
			// 不再处理客户端消息，因为旧日志已经通过 HTTP 接口获取
		}
	}()

	for {
		select {
		case <-ctx.Done():
			return
		case msg, ok := <-messages:
			if !ok {
				return
			}
			if msg.Type == "log_append" {
				if err := conn.WriteJSON(msg.Entry); err != nil {
					cancel()
					return
				}
				continue
			}
			if msg.Type == "resync_required" || msg.Type == "error" {
				level := "warn"
				if msg.Type == "error" {
					level = "error"
				}
				entry := LogEntry{
					Level:     level,
					Message:   "日志流需要重新同步：" + msg.Reason,
					Timestamp: time.Now().Format("2006-01-02 15:04:05.000000"),
				}
				if err := conn.WriteJSON(entry); err != nil {
					cancel()
					return
				}
			}
		}
	}
}
