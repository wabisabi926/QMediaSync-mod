package controllers

import (
	"fmt"

	"qmediasync/internal/models"
)

type netFileCacheStatus string

const (
	netFileCacheHit        netFileCacheStatus = "hit"
	netFileCacheMiss       netFileCacheStatus = "miss"
	netFileCachePartialHit netFileCacheStatus = "partial_hit"
	netFileCacheRefresh    netFileCacheStatus = "refresh"
)

type netFileSourceCapability struct {
	BatchSize  int
	TotalExact bool
}

type netFileSortParams struct {
	V115Order  string
	V115Asc    string
	BaiduOrder string
	BaiduDesc  int32
}

type netFileBatchRange struct {
	Start int
	Size  int
}

type netFileCacheMeta struct {
	Status     netFileCacheStatus `json:"status"`
	BatchStart int                `json:"batch_start"`
	BatchSize  int                `json:"batch_size"`
	CachedAt   int64              `json:"cached_at"`
	ExpiresAt  int64              `json:"expires_at"`
}

type netFileListResponse struct {
	List       []*FileItem      `json:"list"`
	Total      int64            `json:"total"`
	TotalExact bool             `json:"total_exact"`
	HasMore    bool             `json:"has_more"`
	Page       int              `json:"page"`
	PageSize   int              `json:"page_size"`
	SortBy     string           `json:"sort_by"`
	SortOrder  string           `json:"sort_order"`
	Cache      netFileCacheMeta `json:"cache"`
}

func getNetFileSourceCapability(sourceType models.SourceType, sortBy string, sortOrder string) (netFileSourceCapability, error) {
	switch sourceType {
	case models.SourceType115:
		if sortBy == "" {
			sortBy = "name"
		}
		if _, _, err := map115Sort(sortBy, sortOrder); err != nil {
			return netFileSourceCapability{}, err
		}
		return netFileSourceCapability{BatchSize: 1000, TotalExact: true}, nil
	case models.SourceTypeBaiduPan:
		if sortBy == "" {
			sortBy = "name"
		}
		if _, _, err := mapBaiduSort(sortBy, sortOrder); err != nil {
			return netFileSourceCapability{}, err
		}
		return netFileSourceCapability{BatchSize: 1000, TotalExact: false}, nil
	case models.SourceTypeOpenList:
		if sortBy == "" {
			sortBy = "default"
		}
		if sortBy != "default" {
			return netFileSourceCapability{}, fmt.Errorf("OpenList 暂不支持排序")
		}
		return netFileSourceCapability{BatchSize: 500, TotalExact: true}, nil
	default:
		return netFileSourceCapability{}, fmt.Errorf("未知的网盘类型")
	}
}

func map115Sort(sortBy string, sortOrder string) (string, string, error) {
	var order string
	switch sortBy {
	case "", "name":
		order = "file_name"
	case "size":
		order = "file_size"
	case "time":
		order = "user_utime"
	case "type":
		order = "file_type"
	default:
		return "", "", fmt.Errorf("115 不支持排序字段：%s", sortBy)
	}
	asc := "1"
	if sortOrder == "desc" {
		asc = "0"
	}
	return order, asc, nil
}

func mapBaiduSort(sortBy string, sortOrder string) (string, int32, error) {
	var order string
	switch sortBy {
	case "", "name":
		order = "name"
	case "size":
		order = "size"
	case "time":
		order = "time"
	default:
		return "", 0, fmt.Errorf("百度网盘不支持排序字段：%s", sortBy)
	}
	var desc int32
	if sortOrder == "desc" {
		desc = 1
	}
	return order, desc, nil
}

func computeNetFileBatchRanges(page int, pageSize int, batchSize int) []netFileBatchRange {
	uiStart := (page - 1) * pageSize
	uiEnd := uiStart + pageSize
	firstBatchStart := (uiStart / batchSize) * batchSize
	ranges := make([]netFileBatchRange, 0, 2)
	for start := firstBatchStart; start < uiEnd; start += batchSize {
		ranges = append(ranges, netFileBatchRange{Start: start, Size: batchSize})
	}
	return ranges
}

func buildBaiduSyntheticTotal(batchStart int, itemCount int, batchSize int) (int64, bool) {
	if itemCount >= batchSize {
		return int64(batchStart + itemCount + 1), true
	}
	return int64(batchStart + itemCount), false
}

func buildNetFileListResponse(list []*FileItem, total int64, page, pageSize int) netFileListResponse {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = len(list)
	}
	loadedTotal := int64((page-1)*pageSize + len(list))
	if total < loadedTotal {
		total = loadedTotal
	}
	return netFileListResponse{
		List:      list,
		Total:     total,
		Page:      page,
		PageSize:  pageSize,
		SortOrder: "asc",
	}
}
