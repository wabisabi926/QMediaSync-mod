package requests

import "testing"

func TestUpdateThreadsRequestValidate(t *testing.T) {
	valid := UpdateThreadsRequest{
		DownloadThreads:    1,
		FileDetailThreads:  2,
		OpenlistQPS:        2,
		OpenlistRetry:      1,
		OpenlistRetryDelay: 30,
		FileListPageSize:   1150,
	}

	tests := []struct {
		name    string
		mutate  func(*UpdateThreadsRequest)
		wantErr bool
	}{
		{name: "合法线程配置通过"},
		{name: "下载 QPS 为 0 失败", mutate: func(r *UpdateThreadsRequest) { r.DownloadThreads = 0 }, wantErr: true},
		{name: "网盘详情 QPS 小于 2 失败", mutate: func(r *UpdateThreadsRequest) { r.FileDetailThreads = 1 }, wantErr: true},
		{name: "OpenList QPS 大于 10 失败", mutate: func(r *UpdateThreadsRequest) { r.OpenlistQPS = 11 }, wantErr: true},
		{name: "重试间隔小于 30 失败", mutate: func(r *UpdateThreadsRequest) { r.OpenlistRetryDelay = 29 }, wantErr: true},
		{name: "分页数量大于 1150 失败", mutate: func(r *UpdateThreadsRequest) { r.FileListPageSize = 1151 }, wantErr: true},
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

func TestUpdateStrmConfigRequestValidate(t *testing.T) {
	valid := UpdateStrmConfigRequest{
		LocalProxy:     0,
		StrmBaseURL:    "http://127.0.0.1:8096",
		Cron:           "0 2 * * *",
		MinVideoSize:   0,
		VideoExtArr:    []string{".mp4", ".mkv"},
		MetaExtArr:     []string{".nfo", ".jpg"},
		ExcludeNameArr: []string{},
		UploadMeta:     0,
		DownloadMeta:   1,
		DeleteDir:      1,
		AddPath:        2,
		CheckMetaMtime: 0,
	}

	tests := []struct {
		name    string
		mutate  func(*UpdateStrmConfigRequest)
		wantErr bool
	}{
		{name: "合法 STRM 配置通过"},
		{name: "全局 STRM 允许完整路径", mutate: func(r *UpdateStrmConfigRequest) { r.AddPath = 1 }},
		{name: "全局 STRM 允许只添加文件名", mutate: func(r *UpdateStrmConfigRequest) { r.AddPath = 2 }},
		{name: "全局 STRM 允许不添加路径", mutate: func(r *UpdateStrmConfigRequest) { r.AddPath = 3 }},
		{name: "全局 STRM 路径模式枚举错误失败", mutate: func(r *UpdateStrmConfigRequest) { r.AddPath = -1 }, wantErr: true},
		{name: "URL 缺少协议失败", mutate: func(r *UpdateStrmConfigRequest) { r.StrmBaseURL = "127.0.0.1:8096" }, wantErr: true},
		{name: "Cron 格式错误失败", mutate: func(r *UpdateStrmConfigRequest) { r.Cron = "bad" }, wantErr: true},
		{name: "最小视频大小为负数失败", mutate: func(r *UpdateStrmConfigRequest) { r.MinVideoSize = -1 }, wantErr: true},
		{name: "下载元数据枚举错误失败", mutate: func(r *UpdateStrmConfigRequest) { r.DownloadMeta = 2 }, wantErr: true},
		{name: "视频扩展名缺少点失败", mutate: func(r *UpdateStrmConfigRequest) { r.VideoExtArr = []string{"mp4"} }, wantErr: true},
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
