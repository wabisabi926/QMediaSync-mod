package synccron

import (
	"os"
	"strings"
	"testing"
	"time"

	"qmediasync/internal/models"
)

func TestEmbyCronUsesScheduledSyncModeSelector(t *testing.T) {
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
	if !strings.Contains(embyCronBlock, "selectScheduledEmbySyncMode") {
		t.Fatal("Emby 定时任务应通过同步模式选择函数决定执行全量或增量")
	}
	if !strings.Contains(embyCronBlock, "emby.PerformEmbyIncrementalSync()") {
		t.Fatal("Emby 定时任务应保留增量同步分支")
	}
	if !strings.Contains(embyCronBlock, "emby.PerformEmbySync()") {
		t.Fatal("Emby 定时任务应支持每日首次全量同步分支")
	}
}

func TestSelectScheduledEmbySyncMode(t *testing.T) {
	loc := time.FixedZone("CST", 8*60*60)
	now := time.Date(2026, 6, 30, 1, 30, 0, 0, loc)
	todayFullSync := time.Date(2026, 6, 30, 0, 5, 0, 0, loc).Unix()
	yesterdayFullSync := time.Date(2026, 6, 29, 23, 50, 0, 0, loc).Unix()

	tests := []struct {
		name   string
		config *models.EmbyConfig
		want   string
	}{
		{
			name: "关闭每日首次全量同步时执行增量",
			config: &models.EmbyConfig{
				EnableDailyFirstFullSync: 0,
				LastFullSyncAt:           0,
			},
			want: models.EmbySyncModeIncremental,
		},
		{
			name: "启用且从未全量同步时执行全量",
			config: &models.EmbyConfig{
				EnableDailyFirstFullSync: 1,
				LastFullSyncAt:           0,
			},
			want: models.EmbySyncModeFull,
		},
		{
			name: "启用且今天尚未全量成功时执行全量",
			config: &models.EmbyConfig{
				EnableDailyFirstFullSync: 1,
				LastFullSyncAt:           yesterdayFullSync,
			},
			want: models.EmbySyncModeFull,
		},
		{
			name: "启用且今天已有全量成功时执行增量",
			config: &models.EmbyConfig{
				EnableDailyFirstFullSync: 1,
				LastFullSyncAt:           todayFullSync,
			},
			want: models.EmbySyncModeIncremental,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := selectScheduledEmbySyncMode(tt.config, now)
			if got != tt.want {
				t.Fatalf("selectScheduledEmbySyncMode() = %q, want %q", got, tt.want)
			}
		})
	}
}
