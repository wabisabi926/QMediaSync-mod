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
