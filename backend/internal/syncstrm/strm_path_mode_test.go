package syncstrm

import "testing"

func TestStrmPathQueryValue(t *testing.T) {
	file := &SyncFileCache{
		Path:     "/电影/华语电影/让子弹飞",
		FileName: "让子弹飞.mp4",
	}

	tests := []struct {
		name string
		mode int
		want string
	}{
		{name: "完整路径", mode: 1, want: "/电影/华语电影/让子弹飞/让子弹飞.mp4"},
		{name: "只添加文件名", mode: 2, want: "让子弹飞.mp4"},
		{name: "不添加路径", mode: 3, want: ""},
		{name: "未知值按不添加处理", mode: 0, want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := strmPathQueryValue(tt.mode, file); got != tt.want {
				t.Fatalf("strmPathQueryValue() = %q，期望 %q", got, tt.want)
			}
		})
	}
}
