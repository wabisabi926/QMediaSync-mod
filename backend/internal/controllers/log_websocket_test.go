package controllers

import (
	"os"
	"strings"
	"testing"
)

func TestLogWebSocketUsesEndCursorWithoutTailScan(t *testing.T) {
	source, err := os.ReadFile("log_websocket.go")
	if err != nil {
		t.Fatalf("读取 log_websocket.go 失败：%v", err)
	}

	content := string(source)
	if !strings.Contains(content, "logstream.ReadEndCursor(fullLogPath)") {
		t.Fatal("实时日志 WebSocket 建连应只读取 EOF cursor")
	}
	if strings.Contains(content, "logstream.ReadTailEntries(fullLogPath, 0)") {
		t.Fatal("实时日志 WebSocket 建连不应通过 ReadTailEntries(fullLogPath, 0) 扫描整个日志文件")
	}
}
