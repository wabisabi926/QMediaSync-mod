package database

import (
	"strings"
	"testing"
)

func TestQuotePostgresIdentifier(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{name: "普通数据库名", input: "qms", want: `"qms"`},
		{name: "下划线数据库名", input: "qms_prod", want: `"qms_prod"`},
		{name: "双引号会被转义", input: `qms"prod`, want: `"qms""prod"`},
		{name: "注入载荷只作为标识符", input: `qms";DROP/**/DATABASE/**/postgres;--`, want: `"qms"";DROP/**/DATABASE/**/postgres;--"`},
		{name: "空数据库名失败", input: "", wantErr: true},
		{name: "超过 PostgreSQL 标识符长度失败", input: strings.Repeat("a", 64), wantErr: true},
		{name: "包含空白失败", input: "qms prod", wantErr: true},
		{name: "包含 NUL 失败", input: "qms\x00prod", wantErr: true},
		{name: "包含控制字符失败", input: "qms\nprod", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := QuotePostgresIdentifier(tt.input)
			if (err != nil) != tt.wantErr {
				t.Fatalf("QuotePostgresIdentifier() error = %v, wantErr %v", err, tt.wantErr)
			}
			if got != tt.want {
				t.Fatalf("QuotePostgresIdentifier() = %q，期望 %q", got, tt.want)
			}
		})
	}
}
