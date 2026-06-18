package helpers

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"

	"gopkg.in/natefinch/lumberjack.v2"
)

var AppLogger *QLogger
var V115Log *QLogger
var OpenListLog *QLogger
var BaiduPanLog *QLogger
var TMDBLog *QLogger

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

func (q *QLogger) Infof(format string, args ...interface{}) {
	q.Logger.Printf("[INFO] "+format, args...)
}

func (q *QLogger) Info(format string) {
	q.Logger.Println("[INFO] " + format)
}

func (q *QLogger) Debugf(format string, args ...interface{}) {
	q.Logger.Printf("[DEBUG] "+format, args...)
}

func (q *QLogger) Debug(format string) {
	q.Logger.Println("[DEBUG] " + format)
}

func (q *QLogger) Errorf(format string, args ...interface{}) {
	q.Logger.Printf("[ERROR] "+format, args...)
}

func (q *QLogger) Error(format string) {
	q.Logger.Println("[ERROR] " + format)
}

func (q *QLogger) Fatalf(format string, args ...interface{}) {
	q.Logger.Fatalf("[FATAL] "+format, args...)
}

func (q *QLogger) Panicf(format string, args ...interface{}) {
	q.Logger.Panicf("[PANIC] "+format, args...)
}

func (q *QLogger) Warnf(format string, args ...interface{}) {
	q.Logger.Printf("[WARN] "+format, args...)
}

func (q *QLogger) Warn(format string) {
	q.Logger.Println("[WARN] " + format)
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
			MaxSize:    10,   // 最大10MB
			MaxBackups: 3,    // 3个备份
			MaxAge:     7,    //days
			Compress:   true, // disabled by default
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
