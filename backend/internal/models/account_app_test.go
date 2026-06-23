package models

import "testing"

func TestIsBuiltIn115AppId(t *testing.T) {
	tests := []struct {
		name   string
		appId  string
		expect bool
	}{
		{name: "QMediaSync", appId: BuiltIn115AppQMediaSync, expect: true},
		{name: "Q115-STRM", appId: BuiltIn115AppQ115STRM, expect: true},
		{name: "MQ的媒体库", appId: BuiltIn115AppMQMediaLibrary, expect: true},
		{name: "自定义应用", appId: "custom-app-id", expect: false},
		{name: "空值", appId: "", expect: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsBuiltIn115AppId(tt.appId); got != tt.expect {
				t.Fatalf("IsBuiltIn115AppId(%q) = %v，期望 %v", tt.appId, got, tt.expect)
			}
		})
	}
}
