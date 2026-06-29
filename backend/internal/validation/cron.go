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
		return New(field, "仅支持 5 位 cron 表达式或 robfig 描述符")
	}
	return nil
}
