package controllers

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestControllersDoNotUseGlobalLoginedUser(t *testing.T) {
	root := filepath.Join("..", "controllers")
	var offenders []string
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return err
		}
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		if strings.Contains(string(data), "LoginedUser") {
			offenders = append(offenders, path)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("扫描控制器失败: %v", err)
	}
	if len(offenders) > 0 {
		t.Fatalf("控制器仍使用 LoginedUser: %v", offenders)
	}
}
