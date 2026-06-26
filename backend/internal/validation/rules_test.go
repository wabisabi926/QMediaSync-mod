package validation

import "testing"

func TestRangeInt(t *testing.T) {
	tests := []struct {
		name    string
		value   int
		min     int
		max     int
		wantErr bool
	}{
		{name: "范围内通过", value: 5, min: 1, max: 10},
		{name: "小于最小值失败", value: 0, min: 1, max: 10, wantErr: true},
		{name: "大于最大值失败", value: 11, min: 1, max: 10, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := RangeInt("download_threads", tt.value, tt.min, tt.max)
			if (err != nil) != tt.wantErr {
				t.Fatalf("RangeInt() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestOneOfInt(t *testing.T) {
	if err := OneOfInt("download_meta", 1, []int{0, 1}); err != nil {
		t.Fatalf("OneOfInt() error = %v", err)
	}
	if err := OneOfInt("download_meta", 2, []int{0, 1}); err == nil {
		t.Fatal("OneOfInt() expected error")
	}
}

func TestHTTPURL(t *testing.T) {
	tests := []struct {
		name       string
		raw        string
		allowEmpty bool
		wantErr    bool
	}{
		{name: "HTTP 通过", raw: "http://127.0.0.1:8096"},
		{name: "HTTPS 通过", raw: "https://emby.example.com"},
		{name: "空值允许", raw: "", allowEmpty: true},
		{name: "空值不允许", raw: "", wantErr: true},
		{name: "缺少协议失败", raw: "127.0.0.1:8096", wantErr: true},
		{name: "FTP 失败", raw: "ftp://example.com", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := HTTPURL("strm_base_url", tt.raw, tt.allowEmpty)
			if (err != nil) != tt.wantErr {
				t.Fatalf("HTTPURL() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestCron(t *testing.T) {
	if err := Cron("cron", "0 2 * * *", false); err != nil {
		t.Fatalf("Cron() error = %v", err)
	}
	if err := Cron("cron", "", true); err != nil {
		t.Fatalf("Cron() allow empty error = %v", err)
	}
	if err := Cron("cron", "invalid", false); err == nil {
		t.Fatal("Cron() expected error")
	}
}

func TestExtList(t *testing.T) {
	if err := ExtList("video_ext_arr", []string{".mp4", ".mkv"}, false); err != nil {
		t.Fatalf("ExtList() error = %v", err)
	}
	if err := ExtList("video_ext_arr", []string{"mp4"}, false); err == nil {
		t.Fatal("ExtList() expected missing dot error")
	}
	if err := ExtList("video_ext_arr", nil, true); err != nil {
		t.Fatalf("ExtList() allow empty error = %v", err)
	}
}
