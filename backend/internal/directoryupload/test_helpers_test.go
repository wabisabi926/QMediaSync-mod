package directoryupload

import (
	"os"
	"path/filepath"
	"testing"
)

func ensureDir(path string) error {
	return os.MkdirAll(path, 0o755)
}

func mustRel(t *testing.T, basePath string, path string) string {
	t.Helper()
	rel, err := filepath.Rel(basePath, path)
	if err != nil {
		t.Fatalf("计算相对路径失败: %v", err)
	}
	return filepath.ToSlash(rel)
}

func assertPathExists(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("路径应存在: %s, err=%v", path, err)
	}
}

func assertPathMissing(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("路径应不存在: %s, err=%v", path, err)
	}
}
