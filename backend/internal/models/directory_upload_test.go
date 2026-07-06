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
	if err := db.Db.AutoMigrate(&SyncPath{}, &SyncFile{}, &EmbyLibrarySyncPath{}, &EmbyMediaSyncFile{}, &DirectoryUploadRule{}, &DirectoryUploadProcessedFile{}); err != nil {
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

func TestDeleteSyncPathByIdDeletesDirectoryUploadRules(t *testing.T) {
	setupDirectoryUploadRuleTestDB(t)

	deletedSyncPath := &SyncPath{
		AccountId:  3,
		SourceType: SourceType115,
		LocalPath:  "/strm/deleted",
		RemotePath: "/remote/deleted",
		BaseCid:    "remote-deleted",
	}
	if err := db.Db.Create(deletedSyncPath).Error; err != nil {
		t.Fatalf("创建待删除同步目录失败: %v", err)
	}
	keptSyncPath := &SyncPath{
		AccountId:  3,
		SourceType: SourceType115,
		LocalPath:  "/strm/kept",
		RemotePath: "/remote/kept",
		BaseCid:    "remote-kept",
	}
	if err := db.Db.Create(keptSyncPath).Error; err != nil {
		t.Fatalf("创建保留同步目录失败: %v", err)
	}

	rules := []*DirectoryUploadRule{
		{
			SyncPathId:     deletedSyncPath.ID,
			AccountId:      3,
			Enabled:        true,
			MonitorPath:    "/watch/deleted",
			RemoteRootPath: "/remote/deleted/uploads",
			RemoteRootId:   "upload-deleted",
		},
		{
			SyncPathId:     keptSyncPath.ID,
			AccountId:      3,
			Enabled:        true,
			MonitorPath:    "/watch/kept",
			RemoteRootPath: "/remote/kept/uploads",
			RemoteRootId:   "upload-kept",
		},
	}
	if err := db.Db.Create(&rules).Error; err != nil {
		t.Fatalf("创建目录监控规则失败: %v", err)
	}

	if ok := DeleteSyncPathById(deletedSyncPath.ID); !ok {
		t.Fatal("删除同步目录应成功")
	}

	var deletedRuleCount int64
	if err := db.Db.Model(&DirectoryUploadRule{}).Where("sync_path_id = ?", deletedSyncPath.ID).Count(&deletedRuleCount).Error; err != nil {
		t.Fatalf("统计被删除同步目录的规则失败: %v", err)
	}
	if deletedRuleCount != 0 {
		t.Fatalf("被删除同步目录的目录监控规则数量 = %d，期望 0", deletedRuleCount)
	}
	var keptRuleCount int64
	if err := db.Db.Model(&DirectoryUploadRule{}).Where("sync_path_id = ?", keptSyncPath.ID).Count(&keptRuleCount).Error; err != nil {
		t.Fatalf("统计保留同步目录的规则失败: %v", err)
	}
	if keptRuleCount != 1 {
		t.Fatalf("保留同步目录的目录监控规则数量 = %d，期望 1", keptRuleCount)
	}
}

func TestDeleteSyncPathByIdDeletesDirectoryUploadProcessedFiles(t *testing.T) {
	setupDirectoryUploadRuleTestDB(t)

	deletedSyncPath := &SyncPath{
		AccountId:  3,
		SourceType: SourceType115,
		LocalPath:  "/strm/deleted",
		RemotePath: "/remote/deleted",
		BaseCid:    "remote-deleted",
	}
	if err := db.Db.Create(deletedSyncPath).Error; err != nil {
		t.Fatalf("创建待删除同步目录失败: %v", err)
	}
	keptSyncPath := &SyncPath{
		AccountId:  3,
		SourceType: SourceType115,
		LocalPath:  "/strm/kept",
		RemotePath: "/remote/kept",
		BaseCid:    "remote-kept",
	}
	if err := db.Db.Create(keptSyncPath).Error; err != nil {
		t.Fatalf("创建保留同步目录失败: %v", err)
	}
	records := []*DirectoryUploadProcessedFile{
		{SyncPathId: deletedSyncPath.ID, SourceKey: "deleted-sync-path-1", SourceFingerprint: "v1:1:1", Result: DirectoryUploadProcessedResultUploaded},
		{SyncPathId: deletedSyncPath.ID, SourceKey: "deleted-sync-path-2", SourceFingerprint: "v1:2:2", Result: DirectoryUploadProcessedResultQueued},
		{SyncPathId: keptSyncPath.ID, SourceKey: "kept-sync-path-1", SourceFingerprint: "v1:3:3", Result: DirectoryUploadProcessedResultRemoteExists},
	}
	if err := db.Db.Create(&records).Error; err != nil {
		t.Fatalf("创建 processed 记录失败: %v", err)
	}

	if ok := DeleteSyncPathById(deletedSyncPath.ID); !ok {
		t.Fatal("删除同步目录应成功")
	}

	var deletedProcessedCount int64
	if err := db.Db.Model(&DirectoryUploadProcessedFile{}).Where("sync_path_id = ?", deletedSyncPath.ID).Count(&deletedProcessedCount).Error; err != nil {
		t.Fatalf("统计被删除同步目录的 processed 记录失败: %v", err)
	}
	if deletedProcessedCount != 0 {
		t.Fatalf("被删除同步目录的 processed 记录数量 = %d，期望 0", deletedProcessedCount)
	}
	var keptProcessedCount int64
	if err := db.Db.Model(&DirectoryUploadProcessedFile{}).Where("sync_path_id = ?", keptSyncPath.ID).Count(&keptProcessedCount).Error; err != nil {
		t.Fatalf("统计保留同步目录的 processed 记录失败: %v", err)
	}
	if keptProcessedCount != 1 {
		t.Fatalf("保留同步目录的 processed 记录数量 = %d，期望 1", keptProcessedCount)
	}
}

func TestDeleteDirectoryUploadRuleDeletesProcessedFiles(t *testing.T) {
	setupDirectoryUploadRuleTestDB(t)

	rules := []*DirectoryUploadRule{
		{
			SyncPathId:     1,
			AccountId:      3,
			Enabled:        true,
			MonitorPath:    "/watch/deleted",
			RemoteRootPath: "/remote/deleted",
			RemoteRootId:   "remote-deleted",
		},
		{
			SyncPathId:     2,
			AccountId:      3,
			Enabled:        true,
			MonitorPath:    "/watch/kept",
			RemoteRootPath: "/remote/kept",
			RemoteRootId:   "remote-kept",
		},
	}
	if err := db.Db.Create(&rules).Error; err != nil {
		t.Fatalf("创建目录监控规则失败: %v", err)
	}
	records := []*DirectoryUploadProcessedFile{
		{RuleId: rules[0].ID, SourceKey: "deleted-1", SourceFingerprint: "v1:1:1", Result: DirectoryUploadProcessedResultUploaded},
		{RuleId: rules[0].ID, SourceKey: "deleted-2", SourceFingerprint: "v1:2:2", Result: DirectoryUploadProcessedResultQueued},
		{RuleId: rules[1].ID, SourceKey: "kept-1", SourceFingerprint: "v1:3:3", Result: DirectoryUploadProcessedResultUploaded},
	}
	if err := db.Db.Create(&records).Error; err != nil {
		t.Fatalf("创建 processed 记录失败: %v", err)
	}

	if err := DeleteDirectoryUploadRule(rules[0].ID); err != nil {
		t.Fatalf("删除目录监控规则失败: %v", err)
	}

	var deletedRuleCount int64
	if err := db.Db.Model(&DirectoryUploadRule{}).Where("id = ?", rules[0].ID).Count(&deletedRuleCount).Error; err != nil {
		t.Fatalf("统计被删除规则失败: %v", err)
	}
	if deletedRuleCount != 0 {
		t.Fatalf("被删除规则数量 = %d，期望 0", deletedRuleCount)
	}
	var deletedProcessedCount int64
	if err := db.Db.Model(&DirectoryUploadProcessedFile{}).Where("rule_id = ?", rules[0].ID).Count(&deletedProcessedCount).Error; err != nil {
		t.Fatalf("统计被删除规则的 processed 记录失败: %v", err)
	}
	if deletedProcessedCount != 0 {
		t.Fatalf("被删除规则的 processed 记录数量 = %d，期望 0", deletedProcessedCount)
	}
	var keptProcessedCount int64
	if err := db.Db.Model(&DirectoryUploadProcessedFile{}).Where("rule_id = ?", rules[1].ID).Count(&keptProcessedCount).Error; err != nil {
		t.Fatalf("统计保留规则的 processed 记录失败: %v", err)
	}
	if keptProcessedCount != 1 {
		t.Fatalf("保留规则的 processed 记录数量 = %d，期望 1", keptProcessedCount)
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
