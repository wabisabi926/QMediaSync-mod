package logs

import (
	"Q115-STRM/emby302/util/logs/colors"
	"Q115-STRM/internal/helpers"
	"fmt"
	"time"
)

// Info 输出蓝色 Info 日志
func Info(format string, v ...any) {
	s := fmt.Sprintf("[INFO] "+format, v...)
	helpers.AppLogger.Infof(colors.ToBlue(s))
}

// Success 输出绿色 Success 日志
func Success(format string, v ...any) {
	s := fmt.Sprintf("[SUCCESS] "+format, v...)
	helpers.AppLogger.Infof(colors.ToGreen(s))
}

// Warn 输出黄色 Warn 日志
func Warn(format string, v ...any) {
	s := fmt.Sprintf("[WARN] "+format, v...)
	helpers.AppLogger.Warnf(colors.ToYellow(s))
}

// Error 输出红色 Error 日志
func Error(format string, v ...any) {
	s := fmt.Sprintf("[ERROR] "+format, v...)
	helpers.AppLogger.Errorf(colors.ToRed(s))
}

// Tip 输出灰色 Tip 日志
func Tip(format string, v ...any) {
	s := fmt.Sprintf(format, v...)
	fmt.Println(now() + colors.ToGray(s))
}

// Progress 输出紫色 Progress 日志
func Progress(format string, v ...any) {
	s := fmt.Sprintf(format, v...)
	fmt.Println(now() + colors.ToPurple(s))
}

// now 返回当前时间戳
func now() string {
	return time.Now().Format("2006-01-02 15:04:05") + " "
}
