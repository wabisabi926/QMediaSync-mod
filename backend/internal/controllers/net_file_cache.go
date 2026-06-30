package controllers

import (
	"sync"
	"time"

	"golang.org/x/sync/singleflight"
)

type netFileBatchCacheKey struct {
	SourceType string
	AccountID  uint
	Path       string
	SortBy     string
	SortOrder  string
	Filter     string
	BatchStart int
	BatchSize  int
}

type netFileBatch struct {
	Items      []*FileItem
	Total      int64
	TotalExact bool
	HasMore    bool
	CachedAt   int64
	ExpiresAt  int64
}

type netFileBatchCache struct {
	mu       sync.Mutex
	maxItems int
	ttl      time.Duration
	items    map[netFileBatchCacheKey]netFileBatch
	order    []netFileBatchCacheKey
}

var netFileCache = newNetFileBatchCache(200, 120*time.Second)
var netFileSingleflight singleflight.Group

func newNetFileBatchCache(maxItems int, ttl time.Duration) *netFileBatchCache {
	if maxItems < 1 {
		maxItems = 1
	}
	return &netFileBatchCache{
		maxItems: maxItems,
		ttl:      ttl,
		items:    make(map[netFileBatchCacheKey]netFileBatch),
		order:    make([]netFileBatchCacheKey, 0, maxItems),
	}
}

func (c *netFileBatchCache) Get(key netFileBatchCacheKey, now time.Time) (netFileBatch, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	batch, ok := c.items[key]
	if !ok {
		return netFileBatch{}, false
	}
	if batch.ExpiresAt > 0 && now.Unix() >= batch.ExpiresAt {
		c.deleteLocked(key)
		return netFileBatch{}, false
	}
	return batch, true
}

func (c *netFileBatchCache) Set(key netFileBatchCacheKey, batch netFileBatch, now time.Time) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if _, exists := c.items[key]; !exists {
		c.order = append(c.order, key)
	}
	batch.CachedAt = now.Unix()
	batch.ExpiresAt = now.Add(c.ttl).Unix()
	c.items[key] = batch
	for len(c.items) > c.maxItems && len(c.order) > 0 {
		c.deleteLocked(c.order[0])
	}
}

func (c *netFileBatchCache) InvalidateView(sourceType string, accountID uint, path string, sortBy string, sortOrder string, filter string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	for key := range c.items {
		if key.SourceType == sourceType &&
			key.AccountID == accountID &&
			key.Path == path &&
			key.SortBy == sortBy &&
			key.SortOrder == sortOrder &&
			key.Filter == filter {
			c.deleteLocked(key)
		}
	}
}

func (c *netFileBatchCache) InvalidatePath(sourceType string, accountID uint, path string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	for key := range c.items {
		if key.SourceType == sourceType && key.AccountID == accountID && key.Path == path {
			c.deleteLocked(key)
		}
	}
}

func (c *netFileBatchCache) Len() int {
	c.mu.Lock()
	defer c.mu.Unlock()

	return len(c.items)
}

func (c *netFileBatchCache) deleteLocked(key netFileBatchCacheKey) {
	delete(c.items, key)
	for i, item := range c.order {
		if item == key {
			c.order = append(c.order[:i], c.order[i+1:]...)
			return
		}
	}
}
