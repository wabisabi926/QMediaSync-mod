package controllers

import (
	"testing"

	"qmediasync/internal/models"
)

func TestNetFileSourceCapability(t *testing.T) {
	tests := []struct {
		name       string
		sourceType models.SourceType
		sortBy     string
		wantBatch  int
		wantErr    bool
	}{
		{name: "115 支持类型排序", sourceType: models.SourceType115, sortBy: "type", wantBatch: 1000},
		{name: "百度不支持类型排序", sourceType: models.SourceTypeBaiduPan, sortBy: "type", wantErr: true},
		{name: "OpenList 默认顺序", sourceType: models.SourceTypeOpenList, sortBy: "default", wantBatch: 500},
		{name: "OpenList 第一版不支持 name 排序", sourceType: models.SourceTypeOpenList, sortBy: "name", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			capability, err := getNetFileSourceCapability(tt.sourceType, tt.sortBy, "asc")
			if (err != nil) != tt.wantErr {
				t.Fatalf("getNetFileSourceCapability() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && capability.BatchSize != tt.wantBatch {
				t.Fatalf("BatchSize = %d, want %d", capability.BatchSize, tt.wantBatch)
			}
		})
	}
}

func TestNetFileBatchRanges(t *testing.T) {
	ranges := computeNetFileBatchRanges(3, 200, 500)
	if len(ranges) != 2 {
		t.Fatalf("ranges len = %d, want 2", len(ranges))
	}
	if ranges[0].Start != 0 || ranges[1].Start != 500 {
		t.Fatalf("ranges = %+v, want starts 0 and 500", ranges)
	}
}

func TestBuildBaiduSyntheticTotal(t *testing.T) {
	total, hasMore := buildBaiduSyntheticTotal(0, 1000, 1000)
	if total != 1001 || !hasMore {
		t.Fatalf("full batch total=%d hasMore=%v, want 1001 true", total, hasMore)
	}
	total, hasMore = buildBaiduSyntheticTotal(1000, 30, 1000)
	if total != 1030 || hasMore {
		t.Fatalf("partial batch total=%d hasMore=%v, want 1030 false", total, hasMore)
	}
}
