package helpers

import (
	"fmt"
	"path/filepath"
	"strings"
)

const (
	defaultSyncLogDir        = "logs/sync"
	legacySyncLogRelativeDir = "libs"
)

// SyncLogDir 返回同步任务日志目录，目录始终限制在 logs 下。
func SyncLogDir() string {
	logConfig := LogConfigSnapshot()
	dir := strings.TrimSpace(logConfig.SyncLogDir)
	if dir == "" {
		dir = defaultSyncLogDir
	}
	dir = filepath.Clean(dir)
	if filepath.IsAbs(dir) {
		return filepath.FromSlash(defaultSyncLogDir)
	}

	slashDir := filepath.ToSlash(dir)
	if slashDir == "." || slashDir == ".." || strings.HasPrefix(slashDir, "../") {
		return filepath.FromSlash(defaultSyncLogDir)
	}
	if slashDir == "logs" || strings.HasPrefix(slashDir, "logs/") {
		return filepath.FromSlash(slashDir)
	}
	return filepath.Join("logs", dir)
}

// SyncLogRelativeDir 返回日志接口使用的同步任务日志相对目录。
func SyncLogRelativeDir() string {
	slashDir := filepath.ToSlash(SyncLogDir())
	if slashDir == "logs" {
		return ""
	}
	return strings.TrimPrefix(slashDir, "logs/")
}

// LegacySyncLogRelativeDir 返回历史同步任务日志相对目录。
func LegacySyncLogRelativeDir() string {
	return legacySyncLogRelativeDir
}

// SyncLogFileName 返回同步任务日志文件名。
func SyncLogFileName(syncID uint) string {
	return fmt.Sprintf("sync_%d.log", syncID)
}
