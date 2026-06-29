package synccron

import (
	"os"
	"strings"
	"testing"
)

func TestEmbyCronRunsIncrementalSync(t *testing.T) {
	data, err := os.ReadFile("synccron.go")
	if err != nil {
		t.Fatalf("读取 synccron.go 失败: %v", err)
	}
	source := string(data)
	embyCronStart := strings.Index(source, "if config, err := models.GetEmbyConfig(); err == nil {")
	if embyCronStart < 0 {
		t.Fatal("未找到 Emby 定时任务配置块")
	}
	nextCron := strings.Index(source[embyCronStart:], "\n\tGlobalCron.AddFunc(\"*/2 * * * *\"")
	if nextCron < 0 {
		t.Fatal("未找到 Emby 定时任务配置块结束位置")
	}
	embyCronBlock := source[embyCronStart : embyCronStart+nextCron]
	if !strings.Contains(embyCronBlock, "emby.PerformEmbyIncrementalSync()") {
		t.Fatal("Emby 定时任务应默认执行增量同步")
	}
	if strings.Contains(embyCronBlock, "emby.PerformEmbySync()") {
		t.Fatal("Emby 定时任务不应每次执行全量同步")
	}
}
