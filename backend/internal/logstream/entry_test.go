package logstream

import "testing"

func TestParseLineExtractsLevelTimestampAndMessage(t *testing.T) {
	entry := ParseLine("2025/11/29 12:33:09.530499 [WARN] 文件不存在")
	if entry.Level != "warn" {
		t.Fatalf("level = %s，期望 warn", entry.Level)
	}
	if entry.Timestamp != "2025/11/29 12:33:09.530499" {
		t.Fatalf("timestamp = %s", entry.Timestamp)
	}
	if entry.Message != "文件不存在" {
		t.Fatalf("message = %s", entry.Message)
	}
}
