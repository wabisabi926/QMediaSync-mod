package models

import (
	"qmediasync/internal/db"
	"qmediasync/internal/helpers"
)

const (
	EmbySyncModeIdle           = "idle"
	EmbySyncModeFull           = "full"
	EmbySyncModeIncremental    = "incremental"
	EmbySyncModeWebhook        = "webhook"
	EmbySyncModeRefreshLibrary = "refresh_library"
)

// EmbyConfig 独立的 Emby 配置表
type EmbyConfig struct {
	BaseModel
	EmbyUrl                 string `json:"emby_url" gorm:"type:varchar(500)"`
	EmbyApiKey              string `json:"emby_api_key" gorm:"type:varchar(200)"`
	EnableDeleteNetdisk     int    `json:"enable_delete_netdisk" gorm:"default:0"`
	EnableRefreshLibrary    int    `json:"enable_refresh_library" gorm:"default:0"`
	EnableMediaNotification int    `json:"enable_media_notification" gorm:"default:0"`
	EnableExtractMediaInfo  int    `json:"enable_extract_media_info" gorm:"default:0"`
	EnableAuth              int    `json:"enable_auth" gorm:"default:1"`
	SyncEnabled             int    `json:"sync_enabled" gorm:"default:1"`
	SyncCron                string `json:"sync_cron" gorm:"type:varchar(100);default:'0 * * * *'"`
	LastSyncTime            int64  `json:"last_sync_time" gorm:"default:0"`
	LastFullSyncAt          int64  `json:"last_full_sync_at" gorm:"index;default:0"`
	LastIncrementalSyncAt   int64  `json:"last_incremental_sync_at" gorm:"index;default:0"`
	LastSavedCursorAt       int64  `json:"last_saved_cursor_at" gorm:"index;default:0"`
	LastProcessedCount      int64  `json:"last_processed_count" gorm:"default:0"`
	LastError               string `json:"last_error" gorm:"type:text"`
	IsRunning               bool   `json:"is_running" gorm:"default:false"`
	SyncMode                string `json:"sync_mode" gorm:"size:32;index;default:'idle'"`
	StartedAt               int64  `json:"started_at" gorm:"index;default:0"`
	SelectedLibraries       string `json:"selected_libraries" gorm:"type:text;default:'[]'"` // 选中的媒体库 ID 列表（JSON 格式）
	SyncAllLibraries        int    `json:"sync_all_libraries" gorm:"default:1"`              // 是否同步所有媒体库（1=全部，0=部分）
	EnablePlaybackOverview  int    `json:"enable_playback_overview" gorm:"default:0"`        // 播放通知是否显示剧情简介
	EnablePlaybackProgress  int    `json:"enable_playback_progress" gorm:"default:0"`        // 播放通知是否显示播放进度
	// DeleteNetdiskLibrary    string `json:"delete_netdisk_library" gorm:"type:varchar(200);default:''"` // 允许联动删除的媒体库 ID，用英文逗号分隔，空表示允许全部
}

func (*EmbyConfig) TableName() string {
	return "emby_config"
}

var GlobalEmbyConfig *EmbyConfig

// GetEmbyConfig 获取 Emby 配置
func GetEmbyConfig() (*EmbyConfig, error) {
	if GlobalEmbyConfig != nil {
		return GlobalEmbyConfig, nil
	}
	return GetEmbyConfigFromDB()
}

// GetEmbyConfigFromDB 从数据库读取最新 Emby 配置并刷新内存缓存。
func GetEmbyConfigFromDB() (*EmbyConfig, error) {
	config := &EmbyConfig{}
	if err := db.Db.First(config).Error; err != nil {
		return nil, err
	}
	normalizeEmbySyncMode(config)
	GlobalEmbyConfig = config
	return GlobalEmbyConfig, nil
}

// Update 更新配置
func (c *EmbyConfig) Update(updates map[string]interface{}) error {
	if err := db.Db.Model(c).Updates(updates).Error; err != nil {
		return err
	}
	_, err := GetEmbyConfigFromDB()
	return err
}

// StartEmbySyncRun 标记 Emby 同步任务开始。返回 false 表示已有任务运行。
func StartEmbySyncRun(mode string, startedAt int64) (bool, error) {
	if startedAt <= 0 {
		startedAt = helpers.NowUnix()
	}
	if mode == "" {
		mode = EmbySyncModeFull
	}

	config, err := GetEmbyConfigFromDB()
	if err != nil {
		return false, err
	}
	if config.IsRunning {
		return false, nil
	}

	result := db.Db.Model(&EmbyConfig{}).
		Where("id = ? AND is_running = ?", config.ID, false).
		Updates(map[string]any{
			"is_running": true,
			"sync_mode":  mode,
			"started_at": startedAt,
			"last_error": "",
		})
	if result.Error != nil {
		return false, result.Error
	}
	if result.RowsAffected == 0 {
		return false, nil
	}

	_, err = GetEmbyConfigFromDB()
	return err == nil, err
}

// FinishEmbySyncRun 标记 Emby 同步任务结束，并记录成功或失败状态。
func FinishEmbySyncRun(mode string, processedCount int64, finishedAt int64, runErr error) error {
	if finishedAt <= 0 {
		finishedAt = helpers.NowUnix()
	}
	if mode == "" {
		mode = EmbySyncModeFull
	}

	updates := map[string]any{
		"is_running":           false,
		"sync_mode":            EmbySyncModeIdle,
		"started_at":           0,
		"last_processed_count": processedCount,
	}
	if runErr != nil {
		updates["last_error"] = runErr.Error()
	} else {
		updates["last_error"] = ""
		updates["last_sync_time"] = finishedAt
		switch mode {
		case EmbySyncModeFull:
			updates["last_full_sync_at"] = finishedAt
		case EmbySyncModeIncremental:
			updates["last_incremental_sync_at"] = finishedAt
		case EmbySyncModeWebhook:
			// Webhook 单条同步不推进全量或增量游标。
		}
	}

	if err := db.Db.Model(&EmbyConfig{}).Where("id > 0").Updates(updates).Error; err != nil {
		return err
	}
	_, err := GetEmbyConfigFromDB()
	return err
}

// IsEmbySyncRunningInDB 查询数据库中的 Emby 同步运行状态。
func IsEmbySyncRunningInDB() bool {
	if db.Db == nil {
		return false
	}
	config, err := GetEmbyConfigFromDB()
	return err == nil && config.IsRunning
}

func normalizeEmbySyncMode(config *EmbyConfig) {
	if config.SyncMode == "" {
		config.SyncMode = EmbySyncModeIdle
	}
}
