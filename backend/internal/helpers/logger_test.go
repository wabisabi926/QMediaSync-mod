package helpers

import (
	"bytes"
	"log"
	"strings"
	"testing"
)

func useTestLogLevel(t *testing.T, level LogLevel) {
	t.Helper()
	oldLevel := ConfiguredLogLevel()
	SetGlobalLogLevel(level)
	t.Cleanup(func() {
		SetGlobalLogLevel(oldLevel)
	})
}

func TestRedactSensitiveLog(t *testing.T) {
	input := strings.Join([]string{
		"GET /videos/1/stream?api_key=emby-secret&Static=true",
		"X-Emby-Token=emby-token",
		"Authorization: Bearer auth-secret",
		"X-Emby-Authorization: MediaBrowser Token=\"full-auth-secret\"",
		"X-API-Key: qms-secret",
		"password=db-secret",
		"access_token=access-secret",
		"refresh_token=refresh-secret",
		"AccessKeySecret=aliyun-secret",
		"SecurityToken=security-secret",
		"proxy=http://user:pass@proxy.local:8080",
	}, " ")

	got := RedactSensitiveLog(input)
	secrets := []string{
		"emby-secret",
		"emby-token",
		"auth-secret",
		"full-auth-secret",
		"qms-secret",
		"db-secret",
		"access-secret",
		"refresh-secret",
		"aliyun-secret",
		"security-secret",
	}
	for _, secret := range secrets {
		if strings.Contains(got, secret) {
			t.Fatalf("脱敏结果仍包含敏感值 %q: %s", secret, got)
		}
	}
	if !strings.Contains(got, "******") {
		t.Fatalf("脱敏结果缺少占位符: %s", got)
	}
	if !strings.Contains(got, "http://user:pass@proxy.local:8080") {
		t.Fatalf("普通代理地址不应被脱敏: %s", got)
	}
}

func TestQLogger默认脱敏日志(t *testing.T) {
	useTestLogLevel(t, LogLevelInfo)

	var buf bytes.Buffer
	logger := &QLogger{Logger: log.New(&buf, "", 0)}

	logger.Infof("请求 URI: /videos/1/stream?api_key=%s", "emby-secret")

	got := buf.String()
	if strings.Contains(got, "emby-secret") {
		t.Fatalf("普通日志不应输出敏感值: %s", got)
	}
	if !strings.Contains(got, "******") {
		t.Fatalf("普通日志应输出脱敏占位符: %s", got)
	}
}

func TestRedactSensitiveLogPostgresPasswordWithAmpersand(t *testing.T) {
	input := "连接数据库：host=postgres port=5432 user=postgres password=secret&a#PMTeXv#@rNg8q&d dbname=qmediasync sslmode=disable"

	got := RedactSensitiveLog(input)

	if strings.Contains(got, "secret") || strings.Contains(got, "a#PMTeXv") || strings.Contains(got, "@rNg8q") {
		t.Fatalf("PostgreSQL 密码未完整脱敏: %s", got)
	}
	if !strings.Contains(got, "password=****** dbname=qmediasync") {
		t.Fatalf("PostgreSQL 密码应脱敏为六个星号并保留后续字段: %s", got)
	}
}

func TestQLoggerSensitiveDebugf需要显式开关(t *testing.T) {
	tests := []struct {
		name       string
		envValue   string
		wantSecret bool
	}{
		{name: "默认关闭时脱敏", wantSecret: false},
		{name: "显式开启时保留完整值", envValue: "1", wantSecret: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("QMS_UNSAFE_SENSITIVE_LOG", tt.envValue)
			useTestLogLevel(t, LogLevelDebug)

			var buf bytes.Buffer
			logger := &QLogger{Logger: log.New(&buf, "", 0)}

			logger.SensitiveDebugf("Authorization: Bearer %s", "auth-secret")

			got := buf.String()
			if strings.Contains(got, "auth-secret") != tt.wantSecret {
				t.Fatalf("SensitiveDebugf 输出 = %q，wantSecret %v", got, tt.wantSecret)
			}
		})
	}
}

func TestQLogger按日志等级过滤(t *testing.T) {
	tests := []struct {
		name      string
		level     LogLevel
		want      []string
		doNotWant []string
	}{
		{
			name:      "info 默认写入 info warn error",
			level:     LogLevelInfo,
			want:      []string{"[INFO] info message", "[WARN] warn message", "[ERROR] error message"},
			doNotWant: []string{"[DEBUG] debug message"},
		},
		{
			name:      "debug 写入全部等级",
			level:     LogLevelDebug,
			want:      []string{"[DEBUG] debug message", "[INFO] info message", "[WARN] warn message", "[ERROR] error message"},
			doNotWant: []string{},
		},
		{
			name:      "warn 只写入 warn error",
			level:     LogLevelWarn,
			want:      []string{"[WARN] warn message", "[ERROR] error message"},
			doNotWant: []string{"[DEBUG] debug message", "[INFO] info message"},
		},
		{
			name:      "error 只写入 error",
			level:     LogLevelError,
			want:      []string{"[ERROR] error message"},
			doNotWant: []string{"[DEBUG] debug message", "[INFO] info message", "[WARN] warn message"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			useTestLogLevel(t, tt.level)

			var buf bytes.Buffer
			logger := &QLogger{Logger: log.New(&buf, "", 0)}

			logger.Debug("debug message")
			logger.Info("info message")
			logger.Warn("warn message")
			logger.Error("error message")

			got := buf.String()
			for _, want := range tt.want {
				if !strings.Contains(got, want) {
					t.Fatalf("日志输出缺少 %q: %s", want, got)
				}
			}
			for _, doNotWant := range tt.doNotWant {
				if strings.Contains(got, doNotWant) {
					t.Fatalf("日志输出不应包含 %q: %s", doNotWant, got)
				}
			}
		})
	}
}

func TestQLogger默认日志等级为Info(t *testing.T) {
	useTestLogLevel(t, LogLevelInfo)

	var buf bytes.Buffer
	logger := &QLogger{Logger: log.New(&buf, "", 0)}

	logger.Debug("debug message")
	logger.Info("info message")

	got := buf.String()
	if strings.Contains(got, "[DEBUG] debug message") {
		t.Fatalf("默认日志等级不应写入 Debug: %s", got)
	}
	if !strings.Contains(got, "[INFO] info message") {
		t.Fatalf("默认日志等级应写入 Info: %s", got)
	}
}

func TestSetGlobalLogLevel影响已创建的非全局Logger(t *testing.T) {
	useTestLogLevel(t, LogLevelInfo)

	var buf bytes.Buffer
	logger := &QLogger{Logger: log.New(&buf, "", 0)}

	SetGlobalLogLevel(LogLevelError)
	logger.Info("info message")
	logger.Error("error message")

	got := buf.String()
	if strings.Contains(got, "[INFO] info message") {
		t.Fatalf("全局等级切到 Error 后，已创建的任务 logger 不应继续写入 Info: %s", got)
	}
	if !strings.Contains(got, "[ERROR] error message") {
		t.Fatalf("全局等级切到 Error 后，任务 logger 仍应写入 Error: %s", got)
	}
}

func TestQLoggerSensitiveDebugf遵循日志等级过滤(t *testing.T) {
	t.Setenv("QMS_UNSAFE_SENSITIVE_LOG", "1")
	useTestLogLevel(t, LogLevelInfo)

	var buf bytes.Buffer
	logger := &QLogger{Logger: log.New(&buf, "", 0)}
	logger.SetLevel(LogLevelInfo)

	logger.SensitiveDebugf("Authorization: Bearer %s", "auth-secret")

	if got := buf.String(); got != "" {
		t.Fatalf("Info 等级不应写入 SensitiveDebugf: %s", got)
	}

	logger.SetLevel(LogLevelDebug)
	logger.SensitiveDebugf("Authorization: Bearer %s", "auth-secret")

	if got := buf.String(); !strings.Contains(got, "auth-secret") {
		t.Fatalf("Debug 等级应写入 SensitiveDebugf: %s", got)
	}
}

func TestQLoggerPanicf始终输出(t *testing.T) {
	useTestLogLevel(t, LogLevelError)

	var buf bytes.Buffer
	logger := &QLogger{Logger: log.New(&buf, "", 0)}

	func() {
		defer func() {
			if r := recover(); r == nil {
				t.Fatal("Panicf 未触发 panic")
			}
		}()
		logger.Panicf("panic message")
	}()

	if got := buf.String(); !strings.Contains(got, "[PANIC] panic message") {
		t.Fatalf("Panicf 应忽略日志等级并输出: %s", got)
	}
}
