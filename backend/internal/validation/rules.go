package validation

import (
	"net/url"
	"strings"
)

func NonBlank(field string, value string) error {
	if strings.TrimSpace(value) == "" {
		return New(field, "不能为空")
	}
	return nil
}

func RangeInt(field string, value int, min int, max int) error {
	if value < min || value > max {
		return New(field, "取值超出允许范围")
	}
	return nil
}

func RangeInt64(field string, value int64, min int64, max int64) error {
	if value < min || value > max {
		return New(field, "取值超出允许范围")
	}
	return nil
}

func OneOfInt(field string, value int, allowed []int) error {
	for _, item := range allowed {
		if value == item {
			return nil
		}
	}
	return New(field, "不是允许的取值")
}

func OneOfString(field string, value string, allowed []string) error {
	for _, item := range allowed {
		if value == item {
			return nil
		}
	}
	return New(field, "不是允许的取值")
}

func HTTPURL(field string, raw string, allowEmpty bool) error {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		if allowEmpty {
			return nil
		}
		return New(field, "不能为空")
	}
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Host == "" {
		return New(field, "必须是有效的 HTTP URL")
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return New(field, "只支持 http 或 https")
	}
	return nil
}

func ExtList(field string, values []string, allowEmpty bool) error {
	if len(values) == 0 {
		if allowEmpty {
			return nil
		}
		return New(field, "不能为空")
	}
	for _, value := range values {
		item := strings.TrimSpace(value)
		if item == "" {
			return New(field, "不能包含空值")
		}
		if strings.ContainsAny(item, " \t\r\n") {
			return New(field, "不能包含空白字符")
		}
		if !strings.HasPrefix(item, ".") {
			return New(field, "扩展名必须以 . 开头")
		}
	}
	return nil
}
