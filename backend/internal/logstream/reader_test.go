package logstream

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReadTailEntriesReturnsCursorAtFileEnd(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "app.log")
	content := "2025/11/29 12:33:09.000001 [INFO] one\n2025/11/29 12:33:10.000001 [ERROR] two\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	entries, cursor, err := ReadTailEntries(path, 10)
	if err != nil {
		t.Fatalf("读取日志失败：%v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("entries length = %d，期望 2", len(entries))
	}
	if cursor != int64(len(content)) {
		t.Fatalf("cursor = %d，期望 %d", cursor, len(content))
	}
}
