package helpers

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"gopkg.in/natefinch/lumberjack.v2"
)

var AppLogger *QLogger
var V115Log *QLogger
var OpenListLog *QLogger
var BaiduPanLog *QLogger
var TMDBLog *QLogger

const (
	UnsafeSensitiveLogEnv = "QMS_UNSAFE_SENSITIVE_LOG"
	redactedLogValue      = "[REDACTED]"
)

var (
	sensitiveLogQuotedRegexp           = regexp.MustCompile(`(?i)(["']?(?:api_key|apikey|x-emby-token|authorization|x-emby-authorization|x-api-key|password|access_token|refresh_token|accesskeysecret|securitytoken|cookie|set-cookie)["']?\s*:\s*["'])([^"']*)(["'])`)
	sensitiveLogMediaBrowserAuthRegexp = regexp.MustCompile(`(?i)(\b(?:authorization|x-emby-authorization)\b\s*[:=]\s*MediaBrowser\s+Token=")([^"]*)(")`)
	sensitiveLogAuthRegexp             = regexp.MustCompile(`(?i)(\b(?:authorization|x-emby-authorization)\b\s*[:=]\s*)((?:Bearer|Basic|Token)\s+)?(\[[^\]]*\]|"[^"]*"|'[^']*'|[^\s&,;\]\}]+)`)
	sensitiveLogCookieRegexp           = regexp.MustCompile(`(?i)(\b(?:cookie|set-cookie)\b\s*[:=]\s*)(\[[^\]]*\]|"[^"]*"|'[^']*'|[^,\]\}\n]+)`)
	sensitiveLogKeyValueRegexp         = regexp.MustCompile(`(?i)(\b(?:api_key|apikey|x-emby-token|x-api-key|password|access_token|refresh_token|accesskeysecret|securitytoken)\b\s*[:=]\s*)(\[[^\]]*\]|"[^"]*"|'[^']*'|[^&\s,\]\}]+)`)
)

type QLogger struct {
	*log.Logger
	rotate    bool
	console   bool
	lumLogger *lumberjack.Logger
}

func (q *QLogger) Close() {
	if q.lumLogger != nil {
		q.lumLogger.Close()
	}
}

// RedactSensitiveLog 脱敏日志中的常见密钥、Token 和密码字段。
func RedactSensitiveLog(input string) string {
	if input == "" {
		return input
	}
	output := sensitiveLogQuotedRegexp.ReplaceAllString(input, "${1}"+redactedLogValue+"${3}")
	output = sensitiveLogMediaBrowserAuthRegexp.ReplaceAllString(output, "${1}"+redactedLogValue+"${3}")
	output = sensitiveLogAuthRegexp.ReplaceAllString(output, "${1}${2}"+redactedLogValue)
	output = sensitiveLogCookieRegexp.ReplaceAllString(output, "${1}"+redactedLogValue)
	output = sensitiveLogKeyValueRegexp.ReplaceAllString(output, "${1}"+redactedLogValue)
	return output
}

// UnsafeSensitiveLogEnabled 判断是否允许 unsafe 敏感调试日志输出完整值。
func UnsafeSensitiveLogEnabled() bool {
	value := strings.TrimSpace(os.Getenv(UnsafeSensitiveLogEnv))
	return value == "1" || strings.EqualFold(value, "true") || strings.EqualFold(value, "yes")
}

// WarnUnsafeSensitiveLogIfEnabled 在启用 unsafe 敏感日志时输出显式风险提示。
func WarnUnsafeSensitiveLogIfEnabled() {
	if AppLogger != nil && UnsafeSensitiveLogEnabled() {
		AppLogger.Warnf("%s 已启用，敏感 Debug 日志可能包含 API Key、Token、Cookie 或密码，请勿分享日志文件", UnsafeSensitiveLogEnv)
	}
}

func (q *QLogger) logf(level string, format string, args ...interface{}) {
	q.Logger.Printf("[%s] %s", level, RedactSensitiveLog(fmt.Sprintf(format, args...)))
}

func (q *QLogger) log(level string, message string) {
	q.Logger.Println("[" + level + "] " + RedactSensitiveLog(message))
}

func (q *QLogger) Infof(format string, args ...interface{}) {
	q.logf("INFO", format, args...)
}

func (q *QLogger) Info(format string) {
	q.log("INFO", format)
}

func (q *QLogger) Debugf(format string, args ...interface{}) {
	q.logf("DEBUG", format, args...)
}

func (q *QLogger) Debug(format string) {
	q.log("DEBUG", format)
}

func (q *QLogger) SensitiveDebugf(format string, args ...interface{}) {
	message := fmt.Sprintf(format, args...)
	if !UnsafeSensitiveLogEnabled() {
		message = RedactSensitiveLog(message)
	}
	q.Logger.Printf("[DEBUG] %s", message)
}

func (q *QLogger) SensitiveDebug(format string) {
	message := format
	if !UnsafeSensitiveLogEnabled() {
		message = RedactSensitiveLog(message)
	}
	q.Logger.Println("[DEBUG] " + message)
}

func (q *QLogger) Errorf(format string, args ...interface{}) {
	q.logf("ERROR", format, args...)
}

func (q *QLogger) Error(format string) {
	q.log("ERROR", format)
}

func (q *QLogger) Fatalf(format string, args ...interface{}) {
	q.Logger.Fatalf("[FATAL] %s", RedactSensitiveLog(fmt.Sprintf(format, args...)))
}

func (q *QLogger) Panicf(format string, args ...interface{}) {
	q.Logger.Panicf("[PANIC] %s", RedactSensitiveLog(fmt.Sprintf(format, args...)))
}

func (q *QLogger) Warnf(format string, args ...interface{}) {
	q.logf("WARN", format, args...)
}

func (q *QLogger) Warn(format string) {
	q.log("WARN", format)
}

func NewLogger(logFileName string, isConsole bool, rotate bool) *QLogger {
	if IsFnOS {
		// 飞牛环境下不往控制台输出日志
		isConsole = false
	}
	logFile := filepath.Join(ConfigDir, logFileName)
	var lumLogger *lumberjack.Logger
	// 创建多写入器
	var writers []io.Writer

	// 文件写入器
	if rotate {
		lumLogger = &lumberjack.Logger{
			Filename:   logFile,
			MaxSize:    10,   // 最大 10 MB
			MaxBackups: 3,    // 3 个备份
			MaxAge:     7,    // 天
			Compress:   true, // 默认关闭
		}
		if isConsole {
			// 同时写入文件和控制台
			writers = append(writers, lumLogger, os.Stdout)
		} else {
			// 只写入文件
			writers = append(writers, lumLogger)
		}
	} else {
		fd, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			log.Printf("Failed to open log file: %v", err)
			writers = append(writers, os.Stdout)
		} else {
			if isConsole {
				// 同时写入文件和控制台
				writers = append(writers, fd, os.Stdout)
			} else {
				// 只写入文件
				writers = append(writers, fd)
			}
		}
	}
	// 创建多写入器
	multiWriter := io.MultiWriter(writers...)

	// 创建一个新的日志记录器，包含日期、时间和微秒
	logger := log.New(multiWriter, "", log.Ldate|log.Ltime|log.Lmicroseconds)

	return &QLogger{
		Logger:    logger,
		rotate:    rotate,
		console:   isConsole,
		lumLogger: lumLogger,
	}
}

func CloseLogger() {
	if AppLogger != nil && AppLogger.lumLogger != nil {
		AppLogger.lumLogger.Close()
	}
	if V115Log != nil && V115Log.lumLogger != nil {
		V115Log.lumLogger.Close()
	}
	if OpenListLog != nil && OpenListLog.lumLogger != nil {
		OpenListLog.lumLogger.Close()
	}
	if TMDBLog != nil && TMDBLog.lumLogger != nil {
		TMDBLog.lumLogger.Close()
	}
	fmt.Println("已关闭所有日志记录器")
}

func RotateLog() {
	if AppLogger != nil && AppLogger.rotate {
		AppLogger.lumLogger.Rotate()
	}
	if V115Log != nil && V115Log.rotate {
		V115Log.lumLogger.Rotate()
	}
	if OpenListLog != nil && OpenListLog.rotate {
		OpenListLog.lumLogger.Rotate()
	}
	if TMDBLog != nil && TMDBLog.rotate {
		TMDBLog.lumLogger.Rotate()
	}
	fmt.Println("已轮转所有日志文件")
}
