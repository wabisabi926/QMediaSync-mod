package models

import (
	"io"
	"log"
	"testing"

	"qmediasync/internal/db"
	"qmediasync/internal/helpers"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func setupDirectoryUploadRuleTestDB(t *testing.T) {
	t.Helper()
	if helpers.AppLogger == nil {
		helpers.AppLogger = &helpers.QLogger{Logger: log.New(io.Discard, "", 0)}
	}
	testDb, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("打开测试数据库失败: %v", err)
	}
	db.Db = testDb
	if err := db.Db.AutoMigrate(&SyncPath{}, &DirectoryUploadRule{}); err != nil {
		t.Fatalf("迁移测试表失败: %v", err)
	}
}

func TestDirectoryUploadRuleSaveAndDefaults(t *testing.T) {
	setupDirectoryUploadRuleTestDB(t)

	syncPath := &SyncPath{
		BaseModel:  BaseModel{ID: 10},
		AccountId:  3,
		SourceType: SourceType115,
		LocalPath:  "/strm",
		RemotePath: "/remote",
		BaseCid:    "remote-root",
	}
	if err := db.Db.Create(syncPath).Error; err != nil {
		t.Fatalf("创建同步目录失败: %v", err)
	}

	rule := &DirectoryUploadRule{
		SyncPathId:     syncPath.ID,
		AccountId:      syncPath.AccountId,
		Enabled:        true,
		MonitorPath:    "/watch",
		RemoteRootPath: "/remote/uploads",
		RemoteRootId:   "upload-root",
	}
	if err := SaveDirectoryUploadRule(rule); err != nil {
		t.Fatalf("保存目录监控上传规则失败: %v", err)
	}

	enabled, err := GetEnabledDirectoryUploadRules()
	if err != nil {
		t.Fatalf("查询启用规则失败: %v", err)
	}
	if len(enabled) != 1 {
		t.Fatalf("启用规则数量 = %d，期望 1", len(enabled))
	}
	got := enabled[0]
	if got.WatchMode != DirectoryUploadWatchModeAuto {
		t.Fatalf("watch_mode = %s，期望 auto", got.WatchMode)
	}
	if got.StabilitySeconds != 15 || got.StabilityCheckIntervalSeconds != 2 || got.StabilityRequiredCount != 3 {
		t.Fatalf("稳定性默认值 = %+v，期望 15/2/3", got)
	}
	if got.RescanIntervalSeconds != 30 || !got.StartupScanEnabled || got.ProcessedCacheTTLSeconds != 600 {
		t.Fatalf("补偿扫描默认值 = %+v，期望 rescan=30 startup=true ttl=600", got)
	}
	if got.DeleteSourceAfterSuccess {
		t.Fatal("delete_source_after_success 默认应为 false")
	}
	if got.OverwriteMode != DirectoryUploadOverwriteSkipSame {
		t.Fatalf("overwrite_mode = %s，期望 skip_same", got.OverwriteMode)
	}
	if !got.Recursive {
		t.Fatal("recursive 默认应为 true")
	}

	got.DeleteSourceAfterSuccess = true
	if err := SaveDirectoryUploadRule(got); err != nil {
		t.Fatalf("开启删除源文件开关失败: %v", err)
	}
	reloaded, err := GetDirectoryUploadRuleById(got.ID)
	if err != nil {
		t.Fatalf("重新读取规则失败: %v", err)
	}
	if !reloaded.DeleteSourceAfterSuccess {
		t.Fatal("delete_source_after_success 应可保存为 true")
	}
}

func TestDirectoryUploadRuleValidateWithSyncPath(t *testing.T) {
	tests := []struct {
		name     string
		rule     DirectoryUploadRule
		syncPath SyncPath
		wantErr  bool
	}{
		{
			name: "合法规则通过校验",
			rule: DirectoryUploadRule{
				MonitorPath:    "/watch",
				RemoteRootPath: "/remote/uploads",
			},
			syncPath: SyncPath{
				LocalPath:  "/strm",
				RemotePath: "/remote",
			},
		},
		{
			name: "监控目录等于 STRM 本地目录时拒绝",
			rule: DirectoryUploadRule{
				MonitorPath:    "/strm",
				RemoteRootPath: "/remote/uploads",
			},
			syncPath: SyncPath{
				LocalPath:  "/strm",
				RemotePath: "/remote",
			},
			wantErr: true,
		},
		{
			name: "远端上传根目录不在同步远端目录下时拒绝",
			rule: DirectoryUploadRule{
				MonitorPath:    "/watch",
				RemoteRootPath: "/other/uploads",
			},
			syncPath: SyncPath{
				LocalPath:  "/strm",
				RemotePath: "/remote",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.rule.ValidateWithSyncPath(&tt.syncPath)
			if tt.wantErr && err == nil {
				t.Fatal("期望校验失败，实际返回 nil")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("期望校验通过，实际错误: %v", err)
			}
		})
	}
}
