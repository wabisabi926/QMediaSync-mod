package requests

import (
	"testing"

	"qmediasync/internal/models"
)

func TestSaveScrapePathRequestValidate(t *testing.T) {
	valid := SaveScrapePathRequest{
		AccountID:          1,
		SourceType:         models.SourceType115,
		MediaType:          models.MediaTypeMovie,
		SourcePath:         "/movies",
		SourcePathID:       "source-id",
		DestPath:           "/library",
		DestPathID:         "dest-id",
		ScrapeType:         models.ScrapeTypeScrapeAndRename,
		RenameType:         models.RenameTypeMove,
		FolderNameTemplate: "{{title}} ({{year}})",
		FileNameTemplate:   "{{title}}",
		VideoExtList:       []string{".mp4", ".mkv"},
		MaxThreads:         5,
		EnableCron:         true,
		CronExpression:     "0 3 * * *",
	}

	tests := []struct {
		name    string
		mutate  func(*SaveScrapePathRequest)
		wantErr bool
	}{
		{name: "合法网盘刮削路径通过"},
		{name: "非本地缺少账号失败", mutate: func(r *SaveScrapePathRequest) { r.AccountID = 0 }, wantErr: true},
		{name: "未知媒体类型失败", mutate: func(r *SaveScrapePathRequest) { r.MediaType = models.MediaType("bad") }, wantErr: true},
		{name: "非本地软链接失败", mutate: func(r *SaveScrapePathRequest) { r.RenameType = models.RenameTypeSoftSymlink }, wantErr: true},
		{name: "非本地线程大于 5 失败", mutate: func(r *SaveScrapePathRequest) { r.MaxThreads = 6 }, wantErr: true},
		{name: "本地线程 20 通过", mutate: func(r *SaveScrapePathRequest) {
			r.SourceType = models.SourceTypeLocal
			r.AccountID = 0
			r.RenameType = models.RenameTypeHardSymlink
			r.MaxThreads = 20
		}},
		{name: "启用 Cron 但表达式为空失败", mutate: func(r *SaveScrapePathRequest) { r.CronExpression = "" }, wantErr: true},
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
