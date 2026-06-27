package helpers

import (
	"bytes"
	"log"
	"strings"
	"testing"
)

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
