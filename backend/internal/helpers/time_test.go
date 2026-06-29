package helpers

import (
	"testing"
	"time"
)

func TestParseRFC3339Unix(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		want    int64
		wantErr bool
	}{
		{
			name:  "UTC 时间",
			value: "2026-06-28T13:31:43Z",
			want:  time.Date(2026, 6, 28, 13, 31, 43, 0, time.UTC).Unix(),
		},
		{
			name:    "非法 RFC3339",
			value:   "2026-06-28 13:31:43",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseRFC3339Unix(tt.value)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("ParseRFC3339Unix(%q) error = nil, want error", tt.value)
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseRFC3339Unix(%q) error = %v", tt.value, err)
			}
			if got != tt.want {
				t.Fatalf("ParseRFC3339Unix(%q) = %d, want %d", tt.value, got, tt.want)
			}
		})
	}
}

func TestUnixToTime(t *testing.T) {
	timestamp := time.Date(2026, 6, 28, 13, 31, 43, 0, time.UTC).Unix()

	got := UnixToTime(timestamp)
	if got.Location() != time.UTC {
		t.Fatalf("UnixToTime(%d) location = %v, want UTC", timestamp, got.Location())
	}
	if got.Format(time.RFC3339) != "2026-06-28T13:31:43Z" {
		t.Fatalf("UnixToTime(%d) = %s, want 2026-06-28T13:31:43Z", timestamp, got.Format(time.RFC3339))
	}
}

func TestNowUnix(t *testing.T) {
	before := time.Now().UTC().Unix()
	got := NowUnix()
	after := time.Now().UTC().Unix()

	if got < before || got > after {
		t.Fatalf("NowUnix() = %d, want between %d and %d", got, before, after)
	}
}

func TestFormatUnixLogTime(t *testing.T) {
	tests := []struct {
		name  string
		value int64
		want  string
	}{
		{
			name:  "有效时间戳",
			value: time.Date(2026, 6, 28, 13, 31, 43, 0, time.Local).Unix(),
			want:  "2026-06-28 13:31:43",
		},
		{
			name:  "零值时间戳",
			value: 0,
			want:  "未设置",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := FormatUnixLogTime(tt.value); got != tt.want {
				t.Fatalf("FormatUnixLogTime(%d) = %q, want %q", tt.value, got, tt.want)
			}
		})
	}
}
