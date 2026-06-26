package validation

import (
	"strings"

	"github.com/robfig/cron/v3"
)

func Cron(field string, expression string, allowEmpty bool) error {
	expression = strings.TrimSpace(expression)
	if expression == "" {
		if allowEmpty {
			return nil
		}
		return New(field, "不能为空")
	}
	if _, err := cron.ParseStandard(expression); err != nil {
		return New(field, "Cron 表达式格式不正确")
	}
	return nil
}
