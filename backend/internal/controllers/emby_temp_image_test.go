package controllers

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func TestCreateEmbyTempImagePath使用受控随机文件名(t *testing.T) {
	cases := []struct {
		name   string
		itemID string
	}{
		{name: "普通 ID", itemID: "12345"},
		{name: "路径穿越 ID", itemID: "../../etc/passwd"},
		{name: "绝对路径 ID", itemID: "/tmp/evil"},
		{name: "包含分隔符 ID", itemID: "movie/season/item"},
	}

	tempDir, err := filepath.Abs(os.TempDir())
	if err != nil {
		t.Fatalf("获取临时目录失败: %v", err)
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			path, err := createEmbyTempImagePath(tc.itemID)
			if err != nil {
				t.Fatalf("createEmbyTempImagePath() error = %v", err)
			}
			defer func() {
				if err := removeEmbyTempImage(path); err != nil && !os.IsNotExist(err) {
					t.Fatalf("清理临时图片失败: %v", err)
				}
			}()

			absPath, err := filepath.Abs(path)
			if err != nil {
				t.Fatalf("获取临时图片绝对路径失败: %v", err)
			}
			if filepath.Dir(absPath) != tempDir {
				t.Fatalf("临时图片目录 = %q, want %q", filepath.Dir(absPath), tempDir)
			}

			sum := sha256.Sum256([]byte(tc.itemID))
			wantPrefix := fmt.Sprintf("qms_emby_%x_", sum)
			base := filepath.Base(absPath)
			if !strings.HasPrefix(base, wantPrefix) {
				t.Fatalf("临时图片文件名 = %q, want prefix %q", base, wantPrefix)
			}
			if matched := regexp.MustCompile(`^qms_emby_[0-9a-f]{64}_.+\.jpg$`).MatchString(base); !matched {
				t.Fatalf("临时图片文件名格式不合法: %q", base)
			}
		})
	}
}

func TestRemoveEmbyTempImage只删除受控临时图片(t *testing.T) {
	allowed, err := createEmbyTempImagePath("safe-item")
	if err != nil {
		t.Fatalf("createEmbyTempImagePath() error = %v", err)
	}
	if err := removeEmbyTempImage(allowed); err != nil {
		t.Fatalf("removeEmbyTempImage() 删除受控临时图片失败: %v", err)
	}
	if _, err := os.Stat(allowed); !os.IsNotExist(err) {
		t.Fatalf("受控临时图片仍存在或 stat 异常: %v", err)
	}

	unsafePath := filepath.Join(os.TempDir(), "not_qms_emby.jpg")
	if err := os.WriteFile(unsafePath, []byte("x"), 0600); err != nil {
		t.Fatalf("创建非受控临时文件失败: %v", err)
	}
	defer os.Remove(unsafePath)

	if err := removeEmbyTempImage(unsafePath); err == nil {
		t.Fatal("removeEmbyTempImage() 删除非受控临时文件 error = nil, want error")
	}
	if _, err := os.Stat(unsafePath); err != nil {
		t.Fatalf("非受控临时文件不应被删除: %v", err)
	}

	outsideDir := t.TempDir()
	outsidePath := filepath.Join(outsideDir, "qms_emby_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa_x.jpg")
	if err := os.WriteFile(outsidePath, []byte("x"), 0600); err != nil {
		t.Fatalf("创建外部临时文件失败: %v", err)
	}
	if err := removeEmbyTempImage(outsidePath); err == nil {
		t.Fatal("removeEmbyTempImage() 删除临时目录外文件 error = nil, want error")
	}
	if _, err := os.Stat(outsidePath); err != nil {
		t.Fatalf("临时目录外文件不应被删除: %v", err)
	}
}
