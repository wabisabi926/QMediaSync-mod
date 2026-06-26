package requests

import (
	"encoding/json"

	"qmediasync/internal/validation"
)

// UpdateEmbyConfigRequest 更新 Emby 配置请求。
type UpdateEmbyConfigRequest struct {
	EmbyURL                 string `json:"emby_url"`
	EmbyAPIKey              string `json:"emby_api_key"`
	EnableDeleteNetdisk     int    `json:"enable_delete_netdisk"`
	EnableRefreshLibrary    int    `json:"enable_refresh_library"`
	EnableMediaNotification int    `json:"enable_media_notification"`
	EnableExtractMediaInfo  int    `json:"enable_extract_media_info"`
	EnableAuth              int    `json:"enable_auth"`
	SyncEnabled             int    `json:"sync_enabled"`
	SyncCron                string `json:"sync_cron"`
	SelectedLibraries       string `json:"selected_libraries"`
	SyncAllLibraries        int    `json:"sync_all_libraries"`
	EnablePlaybackOverview  int    `json:"enable_playback_overview"`
	EnablePlaybackProgress  int    `json:"enable_playback_progress"`
}

// Validate 校验 Emby 配置请求。
func (r UpdateEmbyConfigRequest) Validate() error {
	if err := validation.HTTPURL("emby_url", r.EmbyURL, true); err != nil {
		return err
	}
	if err := validation.Cron("sync_cron", r.SyncCron, true); err != nil {
		return err
	}
	for field, value := range map[string]int{
		"enable_delete_netdisk":     r.EnableDeleteNetdisk,
		"enable_refresh_library":    r.EnableRefreshLibrary,
		"enable_media_notification": r.EnableMediaNotification,
		"enable_extract_media_info": r.EnableExtractMediaInfo,
		"enable_auth":               r.EnableAuth,
		"sync_enabled":              r.SyncEnabled,
		"sync_all_libraries":        r.SyncAllLibraries,
		"enable_playback_overview":  r.EnablePlaybackOverview,
		"enable_playback_progress":  r.EnablePlaybackProgress,
	} {
		if err := validation.OneOfInt(field, value, []int{0, 1}); err != nil {
			return err
		}
	}
	if r.SelectedLibraries != "" && !json.Valid([]byte(r.SelectedLibraries)) {
		return validation.New("selected_libraries", "必须是有效的 JSON 字符串")
	}
	return nil
}
