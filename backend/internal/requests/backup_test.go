package requests

import "testing"

func TestBackupConfigRequestValidate(t *testing.T) {
	valid := BackupConfigUpdateRequest{
		BackupEnabled:   1,
		BackupCron:      "0 4 * * *",
		BackupRetention: 30,
		BackupMaxCount:  10,
		BackupCompress:  1,
	}

	tests := []struct {
		name    string
		mutate  func(*BackupConfigUpdateRequest)
		wantErr bool
	}{
		{name: "合法备份配置通过"},
		{name: "备份开关枚举错误失败", mutate: func(r *BackupConfigUpdateRequest) { r.BackupEnabled = 2 }, wantErr: true},
		{name: "Cron 格式错误失败", mutate: func(r *BackupConfigUpdateRequest) { r.BackupCron = "bad" }, wantErr: true},
		{name: "保留天数为 0 通过", mutate: func(r *BackupConfigUpdateRequest) { r.BackupRetention = 0 }},
		{name: "保留天数大于 365 失败", mutate: func(r *BackupConfigUpdateRequest) { r.BackupRetention = 366 }, wantErr: true},
		{name: "最大数量大于 1000 失败", mutate: func(r *BackupConfigUpdateRequest) { r.BackupMaxCount = 1001 }, wantErr: true},
		{name: "压缩开关枚举错误失败", mutate: func(r *BackupConfigUpdateRequest) { r.BackupCompress = 2 }, wantErr: true},
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
