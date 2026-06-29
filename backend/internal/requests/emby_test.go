package requests

import "testing"

func TestEmbyConfigRequestValidate(t *testing.T) {
	valid := UpdateEmbyConfigRequest{
		EmbyURL:                 "http://127.0.0.1:8096",
		EnableDeleteNetdisk:     1,
		EnableRefreshLibrary:    1,
		EnableMediaNotification: 0,
		EnableExtractMediaInfo:  1,
		EnableAuth:              0,
		SyncEnabled:             1,
		SyncCron:                "0 * * * *",
		SelectedLibraries:       `["movies"]`,
		SyncAllLibraries:        0,
		EnablePlaybackOverview:  1,
		EnablePlaybackProgress:  1,
	}

	tests := []struct {
		name    string
		mutate  func(*UpdateEmbyConfigRequest)
		wantErr bool
	}{
		{name: "合法 Emby 配置通过"},
		{name: "空 URL 允许", mutate: func(r *UpdateEmbyConfigRequest) { r.EmbyURL = "" }},
		{name: "URL 缺少协议失败", mutate: func(r *UpdateEmbyConfigRequest) { r.EmbyURL = "127.0.0.1:8096" }, wantErr: true},
		{name: "同步 Cron 错误失败", mutate: func(r *UpdateEmbyConfigRequest) { r.SyncCron = "bad" }, wantErr: true},
		{name: "开关枚举错误失败", mutate: func(r *UpdateEmbyConfigRequest) { r.EnableAuth = 2 }, wantErr: true},
		{name: "每日首次全量同步开关枚举错误失败", mutate: func(r *UpdateEmbyConfigRequest) {
			value := 2
			r.EnableDailyFirstFullSync = &value
		}, wantErr: true},
		{name: "媒体库 JSON 错误失败", mutate: func(r *UpdateEmbyConfigRequest) { r.SelectedLibraries = "[" }, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := valid
			if tt.mutate != nil {
				tt.mutate(&req)
			}
			err := req.Validate()
			if (err != nil) != tt.wantErr {
				t.Fatalf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
