package helpers

import (
	"fmt"
	"time"
)

// NowUnix 返回当前 UTC Unix 秒。
func NowUnix() int64 {
	return time.Now().UTC().Unix()
}

// ParseRFC3339Unix 将 RFC3339 时间字符串转换为 Unix 秒。
func ParseRFC3339Unix(value string) (int64, error) {
	t, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return 0, err
	}
	return t.UTC().Unix(), nil
}

// UnixToTime 将 Unix 秒转换为 UTC time.Time。
func UnixToTime(value int64) time.Time {
	return time.Unix(value, 0).UTC()
}

func FormatTimestamp(ts int64) string {
	return time.Unix(ts, 0).Format("2006-01-02 15:04:05")
}

// FormatUnixLogTime 将 Unix 秒格式化为日志展示时间。
func FormatUnixLogTime(ts int64) string {
	if ts <= 0 {
		return "未设置"
	}
	return FormatTimestamp(ts)
}

func FormatDuration(seconds int64) string {
	if seconds < 60 {
		return fmt.Sprintf("%d 秒", seconds)
	} else if seconds < 3600 {
		minutes := seconds / 60
		secs := seconds % 60
		return fmt.Sprintf("%d 分 %d 秒", minutes, secs)
	} else {
		hours := seconds / 3600
		minutes := (seconds % 3600) / 60
		secs := seconds % 60
		return fmt.Sprintf("%d 小时 %d 分 %d 秒", hours, minutes, secs)
	}
}
