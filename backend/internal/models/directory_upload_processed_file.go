package models

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	pathpkg "path"
	"path/filepath"
	"strings"
)

// BuildDirectoryUploadSourceFingerprint 生成目录监控源文件签名。
func BuildDirectoryUploadSourceFingerprint(size int64, mtimeNs int64) string {
	return fmt.Sprintf("v1:%d:%d", size, mtimeNs)
}

func directoryUploadHash(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

// BuildDirectoryUploadScopeHash 生成目录监控规则处理范围哈希。
func BuildDirectoryUploadScopeHash(rule *DirectoryUploadRule) string {
	if rule == nil {
		return ""
	}
	monitorPath := filepath.ToSlash(filepath.Clean(rule.MonitorPath))
	remoteRootPath := pathpkg.Clean(strings.ReplaceAll(rule.RemoteRootPath, "\\", "/"))
	remoteRootID := strings.TrimSpace(rule.RemoteRootId)
	raw := fmt.Sprintf(
		"v1\nrule=%d\nsync_path=%d\naccount=%d\nmonitor=%s\nremote_root=%s\nremote_root_id=%s",
		rule.ID,
		rule.SyncPathId,
		rule.AccountId,
		monitorPath,
		remoteRootPath,
		remoteRootID,
	)
	return directoryUploadHash(raw)
}

// BuildDirectoryUploadSourceKey 生成目录监控源文件在规则范围内的稳定键。
func BuildDirectoryUploadSourceKey(scopeHash string, relativePath string) string {
	rel := strings.ReplaceAll(relativePath, "\\", "/")
	rel = filepath.ToSlash(filepath.Clean(rel))
	return directoryUploadHash(scopeHash + "\n" + rel)
}
