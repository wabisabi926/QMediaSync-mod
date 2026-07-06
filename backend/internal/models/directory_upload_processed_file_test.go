package models

import "testing"

func TestBuildDirectoryUploadSourceFingerprintUsesNanosecondMtime(t *testing.T) {
	got := BuildDirectoryUploadSourceFingerprint(1024, 123456789)
	expected := "v1:1024:123456789"
	if got != expected {
		t.Fatalf("BuildDirectoryUploadSourceFingerprint() = %q，期望 %q", got, expected)
	}
}

func TestBuildDirectoryUploadSourceKeyNormalizesRelativePath(t *testing.T) {
	left := BuildDirectoryUploadSourceKey("scope", "Season 01\\Episode.mkv")
	right := BuildDirectoryUploadSourceKey("scope", "Season 01/Episode.mkv")
	if left == "" {
		t.Fatal("BuildDirectoryUploadSourceKey() 不应返回空 key")
	}
	if left != right {
		t.Fatalf("反斜杠和斜杠相对路径生成的 key 不一致：left=%q right=%q", left, right)
	}
}

func TestBuildDirectoryUploadSourceKeyPreservesFilenameSpaces(t *testing.T) {
	tests := []struct {
		name  string
		left  string
		right string
	}{
		{
			name:  "保留文件名前导空格",
			left:  " movie.mkv",
			right: "movie.mkv",
		},
		{
			name:  "保留文件名尾随空格",
			left:  "movie.mkv ",
			right: "movie.mkv",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			left := BuildDirectoryUploadSourceKey("scope", tt.left)
			right := BuildDirectoryUploadSourceKey("scope", tt.right)
			if left == right {
				t.Fatalf("不同空格语义的相对路径生成了相同 key：left_path=%q right_path=%q key=%q", tt.left, tt.right, left)
			}
		})
	}
}
