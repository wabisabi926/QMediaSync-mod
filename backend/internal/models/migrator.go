package models

import (
	"encoding/json"
	"errors"
	"fmt"

	"qmediasync/internal/db"
	"qmediasync/internal/helpers"
	"qmediasync/internal/notification"

	"gorm.io/gorm"
)

type Migrator struct {
	BaseModel
	VersionCode int `json:"version_code"` // 版本号
}

var MaxVersionCode = 60
var AllTables = []any{
	Migrator{},
	BackupConfig{}, BackupRecord{},
	ApiKey{}, UserSession{}, Settings{}, Sync{}, User{}, Account{},
	SyncPath{}, SyncFile{}, DirectoryUploadRule{}, DirectoryUploadProcessedFile{}, SyncPathIdempotencyRecord{},
	Media{}, MediaSeason{}, MediaEpisode{},
	RequestStat{}, EmbyConfig{}, EmbyMediaItem{}, EmbyMediaSyncFile{}, EmbyLibrary{}, EmbyLibrarySyncPath{}, EmbyLibraryRefreshTask{},
	DbDownloadTask{}, DbUploadTask{}, UploadSession{}, StrmGenerationTask{}, NotificationChannel{}, TelegramChannelConfig{}, MeoWChannelConfig{}, BarkChannelConfig{},
	ServerChanChannelConfig{}, CustomWebhookChannelConfig{}, NotificationRule{},
}

func (*Migrator) TableName() string {
	return "migrator"
}

// 数据库迁移
// 如果没有数据则创建
// 如果已有数据库则从数据库中获取版本，根据版本执行变更
func Migrate() {
	// sqliteDb := db.InitSqlite3(dbFile)
	// 先初始化所有表和基础数据
	if !InitDB() {
		// 初始化数据库版本表
		helpers.AppLogger.Info("已完成数据库初始化")
		return
	}
	var migrator Migrator = Migrator{}
	err := db.Db.Model(&migrator).First(&migrator).Error
	if err != nil {
		helpers.AppLogger.Errorf("获取数据库迁移表失败：%v", err)
	}
	db.Db.Statement.PrepareStmt = true
	if migrator.VersionCode == 1 {
		// 数据库版本低于最大版本，需要升级
		db.Db.AutoMigrate(DbDownloadTask{}, DbUploadTask{}, SyncPath{}, Sync{})
		migrator.UpdateVersionCode(db.Db)
	}
	if migrator.VersionCode == 2 {
		// 数据库版本低于最大版本，需要升级
		db.Db.AutoMigrate(SyncFile{})
		migrator.UpdateVersionCode(db.Db)
	}
	if migrator.VersionCode == 3 {
		// 数据库版本低于最大版本，需要升级
		db.Db.AutoMigrate(Account{})
		migrator.UpdateVersionCode(db.Db)
	}
	if migrator.VersionCode == 4 {
		db.Db.AutoMigrate(Media{}, MediaSeason{}, MediaEpisode{})
		migrator.UpdateVersionCode(db.Db)
	}
	if migrator.VersionCode == 5 {
		// 给下载任务添加 m_time 字段
		db.Db.AutoMigrate(DbDownloadTask{})
		migrator.UpdateVersionCode(db.Db)
	}
	if migrator.VersionCode == 6 {
		// 给同步目录增加更多设置
		db.Db.AutoMigrate(SyncPath{})
		// 修改默认值
		updates := map[string]interface{}{
			"delete_dir":     -1,
			"download_meta":  -1,
			"upload_meta":    -1,
			"min_video_size": -1,
		}
		db.Db.Model(&SyncPath{}).Where("id > ?", 0).Updates(updates)
		migrator.UpdateVersionCode(db.Db)
	}
	if migrator.VersionCode == 7 {
		// 给同步目录增加添加路径设置
		db.Db.AutoMigrate(SyncPath{}, Settings{})
		// 修改默认值
		updates := map[string]interface{}{
			"add_path": -1,
		}
		db.Db.Model(&SyncPath{}).Where("id > ?", 0).Updates(updates)
		// 修改配置表默认值
		updates = map[string]interface{}{
			"add_path": 2,
		}
		db.Db.Model(&Settings{}).Where("id > ?", 0).Updates(updates)
		migrator.UpdateVersionCode(db.Db)
	}
	if migrator.VersionCode == 8 {
		// 创建新的通知渠道表
		db.Db.AutoMigrate(
			&NotificationChannel{},
			&TelegramChannelConfig{},
			&MeoWChannelConfig{},
			&BarkChannelConfig{},
			&ServerChanChannelConfig{},
			&NotificationRule{},
		)
		// 迁移现有的 Telegram 设置到新表
		migrateExistingNotificationSettings(db.Db)
		migrator.UpdateVersionCode(db.Db)
	}
	if migrator.VersionCode == 9 {
		// 增加自定义 Webhook 通知渠道表
		db.Db.AutoMigrate(&CustomWebhookChannelConfig{})
		migrator.UpdateVersionCode(db.Db)
	}
	if migrator.VersionCode == 10 {
		// Webhook 渠道配置增加鉴权与 QueryParam 字段
		db.Db.AutoMigrate(&CustomWebhookChannelConfig{})
		migrator.UpdateVersionCode(db.Db)
	}
	if migrator.VersionCode == 11 {
		// 将 account 表的 AppId 字段替换为 AppIdName
		// 查询所有 Account
		// accounts := []Account{}
		// db.Db.Find(&accounts)
		// for _, account := range accounts {
		// appIdName := "自定义"
		// 	switch account.AppId {
		// 	case helpers.GlobalConfig.Open115AppId:
		// 		appIdName = "Q115-STRM"
		// 	case helpers.GlobalConfig.Open115TestAppId:
		// 		appIdName = "MQ的媒体库"
		// 	}
		// 	db.Db.Model(&Account{}).Where("id = ?", account.ID).Update("app_id", appIdName)
		// 	helpers.AppLogger.Infof("Account %d 的 AppId 字段已更新为 AppIdName：%s", account.ID, appIdName)
		// }
		migrator.UpdateVersionCode(db.Db)
	}
	if migrator.VersionCode == 12 {
		// 备份相关表 + Emby 同步相关表
		db.Db.AutoMigrate(
			BackupConfig{}, BackupRecord{},
			EmbyConfig{}, EmbyMediaItem{}, EmbyMediaSyncFile{}, EmbyLibrary{}, EmbyLibrarySyncPath{},
		)
		migrateEmbyConfig(db.Db)
		migrator.UpdateVersionCode(db.Db)
	}
	if migrator.VersionCode == 13 {
		// 备份相关表 + Emby 同步相关表
		db.Db.AutoMigrate(ApiKey{})
		migrator.UpdateVersionCode(db.Db)
	}
	if migrator.VersionCode == 14 {
		// 添加 EnableAuth 字段到 EmbyConfig 表
		db.Db.AutoMigrate(EmbyConfig{})
		migrator.UpdateVersionCode(db.Db)
	}
	if migrator.VersionCode == 15 {
		// 优化 EmbyMediaSyncFile 表，添加 SyncPathId 字段
		db.Db.AutoMigrate(EmbyMediaSyncFile{})
		// 给 EmbyMediaSyncFile 表补充新增的 SyncPathId 字段
		fillSyncPathIdInEmbyMediaSyncFile(db.Db)
		migrator.UpdateVersionCode(db.Db)
	}
	if migrator.VersionCode == 16 {
		// 清空 SyncFile、EmbyMediaSyncFile、DbDownloadTask 表数据
		db.Db.Exec("DELETE FROM sync_files")
		db.Db.Exec("DELETE FROM emby_media_sync_files")
		db.Db.Exec("DELETE FORM db_download_tasks")
		db.Db.AutoMigrate(SyncFile{})
		// 删除已存在的同步缓存表
		db.Db.Exec("DROP TABLE IF EXISTS sync_files_cache")
		migrator.UpdateVersionCode(db.Db)
	}
	if migrator.VersionCode == 17 {
		migrator.UpdateVersionCode(db.Db) // 增加到 18
	}
	if migrator.VersionCode == 18 {
		// 给 User 表添加 IsAdmin 字段
		db.Db.AutoMigrate(SyncFile{})
		migrator.UpdateVersionCode(db.Db)
	}
	if migrator.VersionCode == 19 {
		// 添加 115 请求统计表
		db.Db.AutoMigrate(&RequestStat{})
		migrator.UpdateVersionCode(db.Db)
	}
	if migrator.VersionCode == 20 {
		// 删除不再使用的表
		db.Db.Migrator().DropTable("sync115_path", "sync_files_cache", "backup_task", "restore_task")
		migrator.UpdateVersionCode(db.Db)
	}
	if migrator.VersionCode == 21 {
		db.Db.AutoMigrate(Settings{}) // 增加 OpenList 限速新字段
		// 给新字段添加默认值
		updateData := make(map[string]interface{})
		// 将下载 QPS 默认改为 1，防止限流
		updateData["download_threads"] = 1
		updateData["openlist_qps"] = 2
		updateData["openlist_retry"] = 1
		updateData["openlist_retry_delay"] = 60
		err := db.Db.Model(Settings{}).Where("id >= ?", 1).Updates(updateData).Error
		if err != nil {
			helpers.AppLogger.Errorf("更新 OpenList 限速设置默认值失败：%v", err)
		}
		migrator.UpdateVersionCode(db.Db)
	}
	if migrator.VersionCode == 22 {
		// 给 Settings 表添加 CheckMetaMtime 字段
		db.Db.AutoMigrate(Settings{}, SyncPath{})
		// 默认改为 false
		updateData := make(map[string]int)
		updateData["check_meta_mtime"] = -1
		// 给所有 SyncPath 设置默认值 false
		db.Db.Model(SyncPath{}).Where("id >= ?", 1).Updates(updateData)
		// 给所有 Settings 设置默认值 0
		updateData["check_meta_mtime"] = 0
		db.Db.Model(Settings{}).Where("id >= ?", 1).Updates(updateData)
		migrator.UpdateVersionCode(db.Db)
	}
	if migrator.VersionCode == 23 {
		// 给 Settings 表添加 CheckMetaMtime 字段
		db.Db.AutoMigrate(Settings{}, SyncPath{})
		migrator.UpdateVersionCode(db.Db)
	}
	if migrator.VersionCode == 24 {
		db.Db.AutoMigrate(BackupConfig{}, BackupRecord{})
		// 插入默认配置
		db.Db.Save(&BackupConfig{
			BaseModel:       BaseModel{ID: 1},
			BackupEnabled:   0,
			BackupPath:      "backups",
			BackupRetention: 7,
			BackupMaxCount:  7,
			BackupCompress:  1,
			BackupCron:      "0 2 * * *",
		})
		migrator.UpdateVersionCode(db.Db)
	}
	if migrator.VersionCode == 25 {
		db.Db.AutoMigrate(SyncPath{})
		migrator.UpdateVersionCode(db.Db)
	}
	if migrator.VersionCode == 26 {
		db.Db.AutoMigrate(BackupConfig{}, BackupRecord{}, MediaEpisode{})
		migrator.UpdateVersionCode(db.Db)
	}

	if migrator.VersionCode == 28 {
		db.Db.AutoMigrate(Media{}, MediaEpisode{})
		migrator.UpdateVersionCode(db.Db)
	}
	if migrator.VersionCode == 29 {
		db.Db.AutoMigrate(EmbyLibrarySyncPath{})
		migrator.UpdateVersionCode(db.Db)
	}
	if migrator.VersionCode == 30 {
		// 将 EmbyItem 中的 EmbyData 字段置空
		err := db.Db.Model(EmbyMediaItem{}).Where("id > 0").Update("emby_data", "").Error
		if err != nil {
			helpers.AppLogger.Errorf("更新 EmbyMediaItem 的 EmbyData 字段为空失败：%v", err)
		}
		migrator.UpdateVersionCode(db.Db)
	}
	if migrator.VersionCode == 33 {
		// 为已有渠道添加新的播放通知类型规则（PlaybackStart、PlaybackPause、PlaybackStop）
		addNewNotificationRulesForExistingChannels(db.Db)
		migrator.UpdateVersionCode(db.Db)
	}
	if migrator.VersionCode == 34 {
		// 给 EmbyMediaItem 表添加 ItemIdInt 字段
		db.Db.AutoMigrate(EmbyMediaItem{})
		// 更新所有 item_id_int 字段
		// 每次取 100 个
		var items []*EmbyMediaItem
		page := 1
		helpers.AppLogger.Infof("开始更新 EmbyMediaItem 的 item_id_int 字段")
		for {
			if err := db.Db.Model(EmbyMediaItem{}).Limit(100).Offset((page - 1) * 100).Order("id ASC").Select("id, item_id, item_id_int").Find(&items).Error; err != nil {
				helpers.AppLogger.Errorf("查询 EmbyMediaItem 的 item_id_int 字段失败：%v", err)
			}
			if len(items) == 0 {
				helpers.AppLogger.Warnf("查询 EmbyMediaItem 的 item_id 字段，共 %d 条", len(items))
				break
			}
			// 更新 item_id_int 字段
			for _, item := range items {
				if item.ItemIdInt != 0 {
					continue
				}
				itemIdInt := helpers.StringToInt64(item.ItemId)
				if err := db.Db.Model(EmbyMediaItem{}).Where("id = ?", item.ID).Update("item_id_int", itemIdInt).Error; err != nil {
					helpers.AppLogger.Errorf("更新 EmbyMediaItem 的 item_id_int 字段 \"%s\" => %d 失败：%v", item.ItemId, itemIdInt, err)
				} else {
					helpers.AppLogger.Infof("更新 EmbyMediaItem 的 item_id_int 字段 \"%s\" => %d 成功", item.ItemId, itemIdInt)
				}
			}
			if len(items) < 100 {
				break
			}
			page++
		}
		helpers.AppLogger.Infof("更新 EmbyMediaItem 的 item_id_int 字段完成")
		migrator.UpdateVersionCode(db.Db)
	}
	if migrator.VersionCode == 35 {
		db.Db.AutoMigrate(EmbyConfig{})
		migrator.UpdateVersionCode(db.Db)
	}
	if migrator.VersionCode == 36 {
		// 添加 115 文件列表每页查询数量字段到 Settings 表
		db.Db.AutoMigrate(Settings{})
		helpers.AppLogger.Info("已添加 file_list_page_size 字段到 Settings 表")
		migrator.UpdateVersionCode(db.Db)
	}
	if migrator.VersionCode == 37 {
		// 添加播放通知剧情简介和播放进度开关到 emby_config 表
		db.Db.AutoMigrate(EmbyConfig{})
		helpers.AppLogger.Info("已添加 enable_playback_overview 和 enable_playback_progress 字段到 emby_config 表")
		migrator.UpdateVersionCode(db.Db)
	}

	if migrator.VersionCode == 38 {
		// 添加刮削失败通知类型到 emby_config 表
		addNewNotificationRulesForExistingChannels(db.Db)
		helpers.AppLogger.Info("已添加刮削整理失败通知类型")
		migrator.UpdateVersionCode(db.Db)
	}

	if migrator.VersionCode == 39 {
		// 添加自定义开放平台应用名字段到 account 表
		db.Db.AutoMigrate(Account{})
		helpers.AppLogger.Info("已添加 account.app_id_name 字段")
		migrator.UpdateVersionCode(db.Db)
	}
	if migrator.VersionCode == 40 {
		// 添加 115 授权来源类型和 provider 字段到 account 表
		db.Db.AutoMigrate(Account{})
		helpers.AppLogger.Info("已添加 account.auth_source_type 和 account.auth_provider 字段")
		migrator.UpdateVersionCode(db.Db)
	}
	if migrator.VersionCode == 41 {
		// 添加两步验证和队列重试字段
		db.Db.AutoMigrate(User{}, DbDownloadTask{}, DbUploadTask{})
		helpers.AppLogger.Info("已添加两步验证和队列重试字段")
		migrator.UpdateVersionCode(db.Db)
	}
	if migrator.VersionCode == 42 {
		// 添加 Emby 媒体库刷新任务表
		db.Db.AutoMigrate(EmbyLibraryRefreshTask{})
		helpers.AppLogger.Info("已添加 emby_library_refresh_tasks 表")
		migrator.UpdateVersionCode(db.Db)
	}
	if migrator.VersionCode == 43 {
		// 将任务来源字段从展示文案迁移为稳定存储值
		if err := db.Db.Transaction(func(tx *gorm.DB) error {
			if err := migrateTaskSourceEnumValues(tx); err != nil {
				return err
			}
			nextVersion := migrator.VersionCode + 1
			if err := tx.Model(&migrator).Update("version_code", nextVersion).Error; err != nil {
				return fmt.Errorf("更新迁移版本失败：%w", err)
			}
			migrator.VersionCode = nextVersion
			return nil
		}); err != nil {
			helpers.AppLogger.Errorf("迁移任务来源枚举存储值失败：%v", err)
			return
		}
		helpers.AppLogger.Info("已迁移任务来源枚举存储值")
		helpers.AppLogger.Infof("同步库结构更新完毕，当前数据库版本：%d", migrator.VersionCode)
	}
	if migrator.VersionCode == 44 {
		// 添加可撤销登录会话表
		db.Db.AutoMigrate(UserSession{})
		helpers.AppLogger.Info("已添加 user_sessions 表")
		migrator.UpdateVersionCode(db.Db)
	}
	if migrator.VersionCode == 45 {
		if err := migrateNotificationChannelTypeIndex(db.Db); err != nil {
			helpers.AppLogger.Errorf("迁移通知渠道类型索引失败：%v", err)
			return
		}
		addMissingNotificationRulesForExistingChannels(db.Db)
		migrator.UpdateVersionCode(db.Db)
	}
	if migrator.VersionCode == 46 {
		if err := db.Db.AutoMigrate(User{}); err != nil {
			helpers.AppLogger.Errorf("迁移用户单用户约束失败：%v", err)
			return
		}
		helpers.AppLogger.Info("已添加 users.singleton_key 单用户约束")
		migrator.UpdateVersionCode(db.Db)
	}
	if migrator.VersionCode == 47 {
		helpers.AppLogger.Info("迁移 STRM 链接路径模式：旧值 2（不添加路径）改为新值 3")
		if err := db.Db.Model(&Settings{}).Where("add_path = ?", 2).Update("add_path", 3).Error; err != nil {
			helpers.AppLogger.Errorf("迁移 settings.add_path 失败：%v", err)
			return
		}
		if err := db.Db.Model(&SyncPath{}).Where("add_path = ?", 2).Update("add_path", 3).Error; err != nil {
			helpers.AppLogger.Errorf("迁移 sync_paths.add_path 失败：%v", err)
			return
		}
		migrator.UpdateVersionCode(db.Db)
	}
	if migrator.VersionCode == 48 {
		if err := db.Db.AutoMigrate(DbDownloadTask{}); err != nil {
			helpers.AppLogger.Errorf("迁移下载任务同步目录字段失败：%v", err)
			return
		}
		helpers.AppLogger.Info("已添加 db_download_tasks.sync_path_id 字段")
		migrator.UpdateVersionCode(db.Db)
	}
	if migrator.VersionCode == 49 {
		if err := db.Db.AutoMigrate(EmbyConfig{}, EmbyMediaItem{}); err != nil {
			helpers.AppLogger.Errorf("迁移 Emby 同步状态和全量批次字段失败：%v", err)
			return
		}
		helpers.AppLogger.Info("已添加 Emby 同步状态和全量同步批次字段")
		migrator.UpdateVersionCode(db.Db)
	}
	if migrator.VersionCode == 50 {
		lastSuccessSyncMode, err := inferExistingEmbyLastSuccessSyncMode(db.Db)
		if err != nil {
			helpers.AppLogger.Errorf("读取 Emby 最近成功同步模式失败：%v", err)
			return
		}
		if err := db.Db.AutoMigrate(EmbyConfig{}); err != nil {
			helpers.AppLogger.Errorf("迁移 Emby 每日首次全量同步字段失败：%v", err)
			return
		}
		if err := db.Db.Model(&EmbyConfig{}).
			Where("enable_daily_first_full_sync = ?", 0).
			Update("enable_daily_first_full_sync", 1).Error; err != nil {
			helpers.AppLogger.Errorf("初始化 Emby 每日首次全量同步开关失败：%v", err)
			return
		}
		if err := backfillEmbyLastSuccessSyncMode(db.Db, lastSuccessSyncMode); err != nil {
			helpers.AppLogger.Errorf("回填 Emby 最近成功同步模式失败：%v", err)
			return
		}
		helpers.AppLogger.Info("已添加 Emby 每日首次全量同步和最近成功模式字段")
		migrator.UpdateVersionCode(db.Db)
	}
	if migrator.VersionCode == 51 {
		if err := db.Db.AutoMigrate(UploadSession{}, DirectoryUploadRule{}, StrmGenerationTask{}, DbUploadTask{}, Settings{}); err != nil {
			helpers.AppLogger.Errorf("迁移上传会话和 STRM 生成任务模型失败：%v", err)
			return
		}
		helpers.AppLogger.Info("已添加上传会话、目录监控上传规则和 STRM 生成任务模型")
		migrator.UpdateVersionCode(db.Db)
	}
	if migrator.VersionCode == 52 {
		if err := db.Db.AutoMigrate(EmbyLibraryRefreshTask{}); err != nil {
			helpers.AppLogger.Errorf("迁移 Emby 定向刷新任务字段失败：%v", err)
			return
		}
		helpers.AppLogger.Info("已添加 Emby 定向刷新任务字段")
		migrator.UpdateVersionCode(db.Db)
	}
	if migrator.VersionCode == 53 {
		if err := db.Db.AutoMigrate(DbUploadTask{}); err != nil {
			helpers.AppLogger.Errorf("迁移上传任务本地 mtime 字段失败：%v", err)
			return
		}
		helpers.AppLogger.Info("已添加上传任务本地 mtime 字段")
		migrator.UpdateVersionCode(db.Db)
	}
	if migrator.VersionCode == 54 {
		if err := db.Db.AutoMigrate(DirectoryUploadRule{}); err != nil {
			helpers.AppLogger.Errorf("迁移目录监控上传元数据开关失败：%v", err)
			return
		}
		helpers.AppLogger.Info("已添加目录监控上传元数据开关")
		migrator.UpdateVersionCode(db.Db)
	}
	if migrator.VersionCode == 55 {
		if err := db.Db.AutoMigrate(StrmGenerationTask{}); err != nil {
			helpers.AppLogger.Errorf("迁移 STRM Webhook 任务刷新与元数据字段失败：%v", err)
			return
		}
		helpers.AppLogger.Info("已添加 STRM Webhook 任务刷新与元数据字段")
		migrator.UpdateVersionCode(db.Db)
	}
	if migrator.VersionCode == 56 {
		if err := db.Db.AutoMigrate(DirectoryUploadProcessedFile{}, DbUploadTask{}); err != nil {
			helpers.AppLogger.Errorf("迁移目录监控源文件处理记录失败：%v", err)
			return
		}
		helpers.AppLogger.Info("已添加目录监控源文件处理记录表")
		migrator.UpdateVersionCode(db.Db)
	}
	if migrator.VersionCode == 57 {
		if err := db.Db.AutoMigrate(SyncPath{}); err != nil {
			helpers.AppLogger.Errorf("迁移同步目录目录监控总开关失败：%v", err)
			return
		}
		if err := backfillDirectoryUploadEnabled(db.Db); err != nil {
			helpers.AppLogger.Errorf("回填同步目录目录监控总开关失败：%v", err)
			return
		}
		helpers.AppLogger.Info("已添加同步目录目录监控总开关")
		migrator.UpdateVersionCode(db.Db)
	}
	if migrator.VersionCode == 58 {
		if err := db.Db.AutoMigrate(Settings{}); err != nil {
			helpers.AppLogger.Errorf("迁移 URL 有效性检查设置失败：%v", err)
			return
		}
		if err := db.Db.Model(&Settings{}).Where("1 = 1").Updates(map[string]any{
			"url_validity_check_enabled":         DefaultURLValidityCheckEnabled,
			"url_validity_check_timeout_seconds": DefaultURLValidityCheckTimeoutSeconds,
		}).Error; err != nil {
			helpers.AppLogger.Errorf("初始化 URL 有效性检查设置失败：%v", err)
			return
		}
		helpers.AppLogger.Info("已添加 115 直链缓存有效性检查设置")
		migrator.UpdateVersionCode(db.Db)
	}
	if migrator.VersionCode == 59 {
		if err := db.Db.AutoMigrate(SyncPathIdempotencyRecord{}); err != nil {
			helpers.AppLogger.Errorf("迁移同步目录幂等记录失败：%v", err)
			return
		}
		if err := migrateEmbyLibraryRefreshTaskKeys(db.Db); err != nil {
			helpers.AppLogger.Errorf("迁移 Emby 刷新任务唯一键失败：%v", err)
			return
		}
		helpers.AppLogger.Info("已添加同步目录幂等记录和 Emby 刷新任务键")
		migrator.UpdateVersionCode(db.Db)
	}
	helpers.AppLogger.Infof("当前数据库版本 %d", migrator.VersionCode)
}

// migrateEmbyLibraryRefreshTaskKeys 将 item 刷新任务的去重键从 library_id 拆分到 task_key。
func migrateEmbyLibraryRefreshTaskKeys(dbConn *gorm.DB) error {
	if !dbConn.Migrator().HasTable(&EmbyLibraryRefreshTask{}) {
		return nil
	}
	if !dbConn.Migrator().HasColumn(&EmbyLibraryRefreshTask{}, "TaskKey") {
		if err := dbConn.Migrator().AddColumn(&EmbyLibraryRefreshTask{}, "TaskKey"); err != nil {
			return fmt.Errorf("添加 emby_library_refresh_tasks.task_key 失败：%w", err)
		}
	}

	var tasks []EmbyLibraryRefreshTask
	if err := dbConn.Order("id ASC").Find(&tasks).Error; err != nil {
		return fmt.Errorf("读取 Emby 刷新任务失败：%w", err)
	}
	for i := range tasks {
		task := &tasks[i]
		taskKey := embyLibraryRefreshTaskKey(task.LibraryId)
		if task.TargetType == EmbyLibraryRefreshTargetTypeItem {
			taskKey = task.LibraryId
			if len(taskKey) < len("item:") || taskKey[:len("item:")] != "item:" {
				itemIDs := task.GetItemIds()
				if len(itemIDs) > 0 {
					taskKey = embyItemRefreshTaskKey(itemIDs[0])
				}
			}
		}
		if err := dbConn.Model(task).Update("task_key", taskKey).Error; err != nil {
			return fmt.Errorf("回填 Emby 刷新任务 %d 失败：%w", task.ID, err)
		}
	}

	if dbConn.Migrator().HasIndex(&EmbyLibraryRefreshTask{}, "idx_emby_library_refresh_tasks_library_id") {
		if err := dbConn.Migrator().DropIndex(&EmbyLibraryRefreshTask{}, "idx_emby_library_refresh_tasks_library_id"); err != nil {
			return fmt.Errorf("移除 emby_library_refresh_tasks.library_id 唯一索引失败：%w", err)
		}
	}
	for i := range tasks {
		task := &tasks[i]
		if task.TargetType != EmbyLibraryRefreshTargetTypeItem {
			continue
		}
		if err := dbConn.Model(task).Update("library_id", task.FallbackLibraryId).Error; err != nil {
			return fmt.Errorf("回填 Emby item 刷新任务 %d 的媒体库 ID 失败：%w", task.ID, err)
		}
	}
	if !dbConn.Migrator().HasIndex(&EmbyLibraryRefreshTask{}, "idx_emby_library_refresh_tasks_task_key") {
		if err := dbConn.Migrator().CreateIndex(&EmbyLibraryRefreshTask{}, "idx_emby_library_refresh_tasks_task_key"); err != nil {
			return fmt.Errorf("创建 emby_library_refresh_tasks.task_key 唯一索引失败：%w", err)
		}
	}
	if !dbConn.Migrator().HasIndex(&EmbyLibraryRefreshTask{}, "idx_emby_library_refresh_tasks_library_id") {
		if err := dbConn.Migrator().CreateIndex(&EmbyLibraryRefreshTask{}, "idx_emby_library_refresh_tasks_library_id"); err != nil {
			return fmt.Errorf("创建 emby_library_refresh_tasks.library_id 索引失败：%w", err)
		}
	}
	return nil
}

// 补齐缺失的表、字段和索引
func BatchCreateTable() error {
	db.Db.Statement.PrepareStmt = true

	var err error
	var lastErr error
	for _, table := range AllTables {
		err = db.Db.AutoMigrate(table)
		if err != nil {
			lastErr = err
		}
	}
	return lastErr
}

func InitMigrationTable(version int) {
	var migrator Migrator = Migrator{}
	migrator = Migrator{BaseModel: BaseModel{ID: 1}, VersionCode: version} // 初始版本为 version
	db.Db.Save(&migrator)
	helpers.AppLogger.Infof("初始化数据库版本表，当前版本为 %d", version)
}

func InitDB() bool {
	// 初始化
	if db.Db.Migrator().HasTable(Migrator{}) {
		helpers.AppLogger.Info("数据库版本表已存在，跳过初始化数据库过程")
		return true
	}
	BatchCreateTable()
	InitMigrationTable(MaxVersionCode)
	// 初始化默认配置
	InitSettings()
	// 初始化 Emby 配置
	InitEmbyConfig()
	helpers.AppLogger.Info("已完成数据库初始化")
	return false
}

func (m *Migrator) UpdateVersionCode(txOrDb *gorm.DB) {
	m.VersionCode++
	txOrDb.Updates(&m)
	helpers.AppLogger.Infof("同步库结构更新完毕，当前数据库版本：%d", m.VersionCode)
}

func inferExistingEmbyLastSuccessSyncMode(dbConn *gorm.DB) (string, error) {
	type embySyncTimes struct {
		LastFullSyncAt        int64 `gorm:"column:last_full_sync_at"`
		LastIncrementalSyncAt int64 `gorm:"column:last_incremental_sync_at"`
	}
	var times embySyncTimes
	if err := dbConn.Table("emby_config").
		Select("last_full_sync_at, last_incremental_sync_at").
		Limit(1).
		Scan(&times).Error; err != nil {
		return "", err
	}
	switch {
	case times.LastFullSyncAt >= times.LastIncrementalSyncAt && times.LastFullSyncAt > 0:
		return EmbySyncModeFull, nil
	case times.LastIncrementalSyncAt > 0:
		return EmbySyncModeIncremental, nil
	default:
		return "", nil
	}
}

func backfillEmbyLastSuccessSyncMode(dbConn *gorm.DB, fallbackMode string) error {
	var configs []EmbyConfig
	if err := dbConn.Find(&configs).Error; err != nil {
		return err
	}
	for _, config := range configs {
		if config.LastSuccessSyncMode != "" {
			continue
		}
		mode := ""
		switch {
		case config.LastFullSyncAt >= config.LastIncrementalSyncAt && config.LastFullSyncAt > 0:
			mode = EmbySyncModeFull
		case config.LastIncrementalSyncAt > 0:
			mode = EmbySyncModeIncremental
		}
		if mode == "" {
			mode = fallbackMode
		}
		if mode == "" {
			continue
		}
		if err := dbConn.Model(&EmbyConfig{}).Where("id = ?", config.ID).Update("last_success_sync_mode", mode).Error; err != nil {
			return err
		}
	}
	return nil
}

func backfillDirectoryUploadEnabled(dbConn *gorm.DB) error {
	if !dbConn.Migrator().HasTable(&DirectoryUploadRule{}) {
		return nil
	}
	var syncPathIDs []uint
	if err := dbConn.Model(&DirectoryUploadRule{}).
		Where("enabled = ?", true).
		Distinct().
		Pluck("sync_path_id", &syncPathIDs).Error; err != nil {
		return err
	}
	if len(syncPathIDs) == 0 {
		return nil
	}
	return dbConn.Model(&SyncPath{}).
		Where("id IN ?", syncPathIDs).
		Update("directory_upload_enabled", true).Error
}

func InitSettings() {
	defaultSettings := Settings{}
	serr := db.Db.Model(&Settings{}).First(&defaultSettings).Error
	if !errors.Is(serr, gorm.ErrRecordNotFound) {
		return
	}
	// 插入默认值
	metaExtStr, _ := json.Marshal(helpers.GlobalConfig.Strm.MetaExt)
	videoExtStr, _ := json.Marshal(helpers.GlobalConfig.Strm.VideoExt)
	ipv4, _ := helpers.GetLocalIP()
	defaultSettings = Settings{
		// 设置默认值
		TelegramBotToken: "",
		TelegramChatId:   "",
		HttpProxy:        "",
		SettingStrm: SettingStrm{
			Cron:         helpers.GlobalConfig.Strm.Cron,
			MetaExt:      string(metaExtStr),
			VideoExt:     string(videoExtStr),
			MinVideoSize: helpers.GlobalConfig.Strm.MinVideoSize,
			DeleteDir:    0,
			UploadMeta:   0,
			DownloadMeta: 0,
			StrmBaseUrl:  fmt.Sprintf("http://%s:12333", ipv4),
		},
		SettingThreads: SettingThreads{
			DownloadThreads:    1,
			FileDetailThreads:  3,
			OpenlistQPS:        3,
			OpenlistRetry:      1,
			OpenlistRetryDelay: 60,
		},
		SettingUploadRapidWait: SettingUploadRapidWait{
			UploadRapidWaitEnabled:         0,
			UploadRapidWaitTimeoutSeconds:  0,
			UploadRapidWaitIntervalSeconds: 60,
			UploadRapidWaitMinSize:         0,
			UploadRapidWaitForceSize:       0,
			UploadRapidWaitSkipUpload:      0,
		},
		SettingURLValidityCheck: SettingURLValidityCheck{
			URLValidityCheckEnabled:        DefaultURLValidityCheckEnabled,
			URLValidityCheckTimeoutSeconds: DefaultURLValidityCheckTimeoutSeconds,
		},
	}
	db.Db.Save(&defaultSettings)
	helpers.AppLogger.Info("已默认添加配置")
}

func InitEmbyConfig() {
	embyConfig := &EmbyConfig{
		EmbyUrl:                  "",
		EmbyApiKey:               "",
		SyncEnabled:              0,
		SyncCron:                 "0 * * * *",
		EnableDeleteNetdisk:      0,
		EnableRefreshLibrary:     0,
		EnableMediaNotification:  0,
		EnableExtractMediaInfo:   0,
		EnableAuth:               1,
		EnableDailyFirstFullSync: 1,
		LastSyncTime:             0,
		SyncMode:                 EmbySyncModeIdle,
	}
	db.Db.Save(embyConfig)
	helpers.AppLogger.Info("已默认添加 Emby 配置")

}

func migrateEmbyConfig(dbConn *gorm.DB) {
	var count int64
	if err := dbConn.Model(&EmbyConfig{}).Count(&count).Error; err != nil {
		return
	}
	if count > 0 {
		return
	}
	var settings Settings
	if err := dbConn.First(&settings).Error; err != nil {
		return
	}
	config := &EmbyConfig{
		EmbyUrl:                  settings.EmbyUrl,
		EmbyApiKey:               settings.EmbyApiKey,
		SyncCron:                 settings.Cron,
		SyncMode:                 EmbySyncModeIdle,
		EnableDailyFirstFullSync: 1,
	}
	dbConn.Create(config)
}

// migrateExistingNotificationSettings 迁移现有的通知设置
func migrateExistingNotificationSettings(dbConn *gorm.DB) {
	var settings Settings
	if err := dbConn.First(&settings).Error; err != nil {
		return
	}

	// 如果存在 Telegram 配置，创建新的记录
	if settings.UseTelegram == 1 && settings.TelegramBotToken != "" {
		channel := NotificationChannel{
			ChannelType: "telegram",
			ChannelName: "Telegram Bot",
			IsEnabled:   true,
		}
		if err := dbConn.Create(&channel).Error; err == nil {
			config := TelegramChannelConfig{
				ChannelID: channel.ID,
				BotToken:  settings.TelegramBotToken,
				ChatID:    settings.TelegramChatId,
				ProxyURL:  settings.HttpProxy,
			}
			dbConn.Create(&config)

			// 创建默认规则（所有事件都发送到此渠道）
			for _, eventType := range notification.AllNotificationTypes {
				rule := NotificationRule{
					ChannelID: channel.ID,
					EventType: string(eventType),
					IsEnabled: true,
				}
				dbConn.Create(&rule)
			}
			helpers.AppLogger.Infof("已迁移 Telegram 通知配置到新表")
		}
	}

	// 如果存在 MeoW 配置，创建新的记录
	if settings.MeoWName != "" {
		channel := NotificationChannel{
			ChannelType: "meow",
			ChannelName: "MeoW",
			IsEnabled:   true,
		}
		if err := dbConn.Create(&channel).Error; err == nil {
			config := MeoWChannelConfig{
				ChannelID: channel.ID,
				Nickname:  settings.MeoWName,
				Endpoint:  "http://api.chuckfang.com",
			}
			dbConn.Create(&config)

			// 创建默认规则
			for _, eventType := range notification.AllNotificationTypes {
				rule := NotificationRule{
					ChannelID: channel.ID,
					EventType: string(eventType),
					IsEnabled: true,
				}
				dbConn.Create(&rule)
			}
			helpers.AppLogger.Infof("已迁移 MeoW 通知配置到新表")
		}
	}
}

func migrateNotificationChannelTypeIndex(dbConn *gorm.DB) error {
	if dbConn.Migrator().HasIndex(&NotificationChannel{}, "idx_channel_type") {
		if err := dbConn.Migrator().DropIndex(&NotificationChannel{}, "idx_channel_type"); err != nil {
			return err
		}
	}
	return dbConn.AutoMigrate(&NotificationChannel{})
}

func addMissingNotificationRulesForExistingChannels(dbConn *gorm.DB) {
	var channels []NotificationChannel
	if err := dbConn.Find(&channels).Error; err != nil {
		helpers.AppLogger.Errorf("获取通知渠道失败：%v", err)
		return
	}

	addedCount := 0
	for _, channel := range channels {
		for _, eventType := range notification.AllNotificationTypes {
			var existingRule NotificationRule
			err := dbConn.Where("channel_id = ? AND event_type = ?", channel.ID, string(eventType)).
				First(&existingRule).Error
			if err == nil {
				continue
			}
			if !errors.Is(err, gorm.ErrRecordNotFound) {
				helpers.AppLogger.Errorf("查询渠道 %d 通知规则失败：%v", channel.ID, err)
				continue
			}
			newRule := NotificationRule{
				ChannelID: channel.ID,
				EventType: string(eventType),
				IsEnabled: true,
			}
			if err := dbConn.Create(&newRule).Error; err != nil {
				helpers.AppLogger.Errorf("为渠道 %d 添加通知规则失败：%v", channel.ID, err)
			} else {
				addedCount++
				helpers.AppLogger.Infof("为渠道 %d（%s）添加通知规则：%s", channel.ID, channel.ChannelName, eventType)
			}
		}
	}

	helpers.AppLogger.Infof("数据库迁移完成：已为 %d 个渠道规则补齐通知类型", addedCount)
}

// addNewNotificationRulesForExistingChannels 为已有渠道补齐缺失的通知类型规则。
func addNewNotificationRulesForExistingChannels(dbConn *gorm.DB) {
	addMissingNotificationRulesForExistingChannels(dbConn)
}

func migrateTaskSourceEnumValues(dbConn *gorm.DB) error {
	updates := []struct {
		model    any
		label    string
		column   string
		oldValue string
		newValue string
	}{
		{model: &DbDownloadTask{}, label: "下载任务来源", column: "source", oldValue: "strm同步", newValue: string(DownloadSourceStrm)},
		{model: &DbDownloadTask{}, label: "下载任务来源", column: "source", oldValue: "本地文件", newValue: string(DownloadSourceLocalFile)},
		{model: &DbDownloadTask{}, label: "下载任务来源", column: "source", oldValue: "emby媒体信息提取", newValue: string(DownloadSourceEmbyMedia)},
		{model: &DbDownloadTask{}, label: "下载任务来源类型", column: "source_type", oldValue: "emby媒体信息提取", newValue: string(SourceTypeEmbyMedia)},
		{model: &DbUploadTask{}, label: "上传任务来源", column: "source", oldValue: "strm同步", newValue: string(UploadSourceStrm)},
	}

	for _, update := range updates {
		if err := updateTaskSourceColumn(dbConn, update.model, update.label, update.column, update.oldValue, update.newValue); err != nil {
			return err
		}
	}
	return nil
}

func updateTaskSourceColumn(dbConn *gorm.DB, model any, label string, column string, oldValue string, newValue string) error {
	result := dbConn.Model(model).Where(column+" = ?", oldValue).Update(column, newValue)
	if result.Error != nil {
		return fmt.Errorf("迁移%s失败：%s -> %s：%w", label, oldValue, newValue, result.Error)
	}
	helpers.AppLogger.Infof("迁移%s完成：%s -> %s，影响 %d 条", label, oldValue, newValue, result.RowsAffected)
	return nil
}

func fillSyncPathIdInEmbyMediaSyncFile(dbConn *gorm.DB) {
	limit := 100
	offset := 0
	for {
		var embyMediaSyncFiles []EmbyMediaSyncFile
		dbConn.Model(&EmbyMediaSyncFile{}).Limit(limit).Offset(offset).Find(&embyMediaSyncFiles)
		if len(embyMediaSyncFiles) == 0 {
			break
		}
		for _, embyMediaSyncFile := range embyMediaSyncFiles {
			// 用 ID 查询 SyncFile
			syncFile := GetSyncFileById(embyMediaSyncFile.SyncFileId)
			if syncFile == nil {
				continue
			}
			embyMediaSyncFile.SyncPathId = syncFile.SyncPathId
			dbConn.Save(&embyMediaSyncFile)
			helpers.AppLogger.Infof("为 EmbyMediaSyncFile %d 填充 SyncPathId %d 成功", embyMediaSyncFile.ID, syncFile.SyncPathId)
		}
		offset += limit
	}
}

func BatchDropTable() error {
	var err, lastErr error
	// 删除所有表
	for _, table := range AllTables {
		err = db.Db.Migrator().DropTable(table)
		if err != nil {
			lastErr = err
			helpers.AppLogger.Errorf("删除表失败：%v", err)
		}
	}
	if lastErr != nil {
		return lastErr
	}
	return nil
}

// 批量更新表的主键序列
// 只处理 PostgreSQL 的修复
func BatchRepairTableSeq() error {
	if helpers.GlobalConfig.Db.Engine != "postgres" {
		return nil
	}
	var err, lastErr error
	// 修复所有表
	for _, table := range AllTables {
		tableName := GetTableName(table)
		err = ResetSequence(tableName, "id")
		if err != nil {
			lastErr = err
			helpers.AppLogger.Errorf("修复表 %s 的主键序列失败：%v", tableName, err)
		}
	}
	if lastErr != nil {
		return lastErr
	}
	return nil
}

func ResetSequence(tableName string, columnName string) error {
	var maxId int64
	// 获取当前最大 ID，如果表为空则从 1 开始
	db.Db.Table(tableName).Select(fmt.Sprintf("COALESCE(MAX(%s), 0)", columnName)).Scan(&maxId)
	if maxId == 0 {
		// 如果没有值则不修复
		return nil
	}
	// 重置序列
	sequenceName := fmt.Sprintf("%s_%s_seq", tableName, columnName)
	return db.Db.Exec(fmt.Sprintf("SELECT setval('%s', ?)", sequenceName), maxId).Error
}
