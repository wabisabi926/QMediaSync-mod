package helpers

import (
	"time"

	"github.com/robfig/cron/v3"
)

// 计算cron表达式的下count次执行时间
func GetNextTimeByCronStr(cronStr string, count int) []time.Time {
	schedule, err := cron.ParseStandard(cronStr)
	if err != nil {
		return nil
	}
	var times []time.Time
	var preTime time.Time
	now := time.Now()
	for i := 0; i < count; i++ {
		if preTime.IsZero() {
			preTime = now
		}
		preTime = schedule.Next(preTime)
		times = append(times, preTime)
	}
	return times
}
