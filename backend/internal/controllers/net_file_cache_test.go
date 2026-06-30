package controllers

import (
	"testing"
	"time"
)

func TestNetFileBatchCacheHitAndExpire(t *testing.T) {
	cache := newNetFileBatchCache(2, time.Second)
	key := netFileBatchCacheKey{
		SourceType: "115",
		AccountID:  1,
		Path:       "0",
		SortBy:     "name",
		SortOrder:  "asc",
		BatchStart: 0,
		BatchSize:  1000,
	}
	cache.Set(key, netFileBatch{Items: []*FileItem{{Id: "1", Name: "a"}}}, time.Now())
	if batch, ok := cache.Get(key, time.Now()); !ok || len(batch.Items) != 1 {
		t.Fatalf("cache hit = (%+v,%v), want one item hit", batch, ok)
	}
	if _, ok := cache.Get(key, time.Now().Add(2*time.Second)); ok {
		t.Fatal("expired batch still hit")
	}
}

func TestNetFileBatchCacheInvalidateView(t *testing.T) {
	cache := newNetFileBatchCache(10, time.Minute)
	now := time.Now()
	base := netFileBatchCacheKey{SourceType: "115", AccountID: 1, Path: "0", BatchStart: 0, BatchSize: 1000}
	nameAsc := base
	nameAsc.SortBy = "name"
	nameAsc.SortOrder = "asc"
	timeDesc := base
	timeDesc.SortBy = "time"
	timeDesc.SortOrder = "desc"
	cache.Set(nameAsc, netFileBatch{}, now)
	cache.Set(timeDesc, netFileBatch{}, now)
	cache.InvalidateView("115", 1, "0", "name", "asc", "")
	if _, ok := cache.Get(nameAsc, now); ok {
		t.Fatal("name asc view still exists")
	}
	if _, ok := cache.Get(timeDesc, now); !ok {
		t.Fatal("time desc view should remain")
	}
}

func TestNetFileBatchCacheInvalidatePath(t *testing.T) {
	cache := newNetFileBatchCache(10, time.Minute)
	now := time.Now()
	cache.Set(netFileBatchCacheKey{SourceType: "115", AccountID: 1, Path: "0", SortBy: "name", SortOrder: "asc", BatchStart: 0, BatchSize: 1000}, netFileBatch{}, now)
	cache.Set(netFileBatchCacheKey{SourceType: "115", AccountID: 1, Path: "0", SortBy: "time", SortOrder: "desc", BatchStart: 0, BatchSize: 1000}, netFileBatch{}, now)
	cache.InvalidatePath("115", 1, "0")
	if cache.Len() != 0 {
		t.Fatalf("Len = %d, want 0", cache.Len())
	}
}

func TestNetFileBatchCacheInvalidateViewAdvancesGeneration(t *testing.T) {
	cache := newNetFileBatchCache(10, time.Minute)
	key := netFileBatchCacheKey{
		SourceType: "openlist",
		AccountID:  1,
		Path:       "/",
		SortBy:     "default",
		SortOrder:  "asc",
		Filter:     "none",
		BatchStart: 0,
		BatchSize:  500,
	}

	before := cache.Generation(key)
	cache.InvalidateView(key.SourceType, key.AccountID, key.Path, key.SortBy, key.SortOrder, key.Filter)
	after := cache.Generation(key)
	if after <= before {
		t.Fatalf("generation 未推进：before=%d after=%d", before, after)
	}
}

func TestNetFileBatchCacheSetIfGenerationPreventsStaleWriteAfterRefresh(t *testing.T) {
	cache := newNetFileBatchCache(10, time.Minute)
	now := time.Now()
	key := netFileBatchCacheKey{
		SourceType: "openlist",
		AccountID:  1,
		Path:       "/",
		SortBy:     "default",
		SortOrder:  "asc",
		Filter:     "none",
		BatchStart: 0,
		BatchSize:  500,
	}

	ordinaryGeneration := cache.Generation(key)
	cache.InvalidateView(key.SourceType, key.AccountID, key.Path, key.SortBy, key.SortOrder, key.Filter)
	refreshGeneration := cache.Generation(key)

	if ok := cache.SetIfGeneration(key, netFileBatch{Items: []*FileItem{{Id: "old", Name: "old"}}}, now, ordinaryGeneration); ok {
		t.Fatal("刷新后旧 generation 的普通请求不应写入缓存")
	}
	if ok := cache.SetIfGeneration(key, netFileBatch{Items: []*FileItem{{Id: "new", Name: "new"}}}, now, refreshGeneration); !ok {
		t.Fatal("当前 generation 的刷新请求应写入缓存")
	}

	batch, ok := cache.Get(key, now)
	if !ok {
		t.Fatal("刷新结果未写入缓存")
	}
	if len(batch.Items) != 1 || batch.Items[0].Id != "new" {
		t.Fatalf("缓存结果 = %+v，期望保留刷新结果", batch.Items)
	}
}
