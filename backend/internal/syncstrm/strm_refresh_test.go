package syncstrm

import (
	"testing"

	"qmediasync/internal/models"
)

func TestShouldRequestEmbyLibraryRefresh(t *testing.T) {
	tests := []struct {
		name    string
		newMeta int64
		newStrm int64
		want    bool
	}{
		{name: "生成 STRM 和下载元数据皆为零时不刷新", newMeta: 0, newStrm: 0, want: false},
		{name: "有新增 STRM 时刷新", newMeta: 0, newStrm: 1, want: true},
		{name: "有新增元数据时刷新", newMeta: 1, newStrm: 0, want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := shouldRequestEmbyLibraryRefresh(tt.newMeta, tt.newStrm); got != tt.want {
				t.Fatalf("shouldRequestEmbyLibraryRefresh(%d, %d) = %v，期望 %v", tt.newMeta, tt.newStrm, got, tt.want)
			}
		})
	}
}

func TestSyncStrmCollectedEmbyRefreshTargetsAreDrained(t *testing.T) {
	syncer := &SyncStrm{}
	syncer.appendEmbyRefreshTarget(models.EmbyRefreshTarget{TargetType: models.EmbyRefreshTargetTypeItem, ItemID: "movie-1"})

	got := syncer.drainEmbyRefreshTargets()
	if len(got) != 1 || got[0].ItemID != "movie-1" {
		t.Fatalf("收集的 Emby 刷新目标 = %+v，期望 movie-1", got)
	}
	if remaining := syncer.drainEmbyRefreshTargets(); len(remaining) != 0 {
		t.Fatalf("读取后不应保留刷新目标，实际 = %+v", remaining)
	}
}
