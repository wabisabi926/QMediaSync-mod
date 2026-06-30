package controllers

import (
	"strings"
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

type netFileBatchCacheViewKey struct {
	SourceType string
	AccountID  uint
	Path       string
	SortBy     string
	SortOrder  string
	Filter     string
}

type netFileBatchCachePathKey struct {
	SourceType string
	AccountID  uint
	Path       string
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
	views    map[netFileBatchCacheViewKey]uint64
	paths    map[netFileBatchCachePathKey]uint64
	trees    map[netFileBatchCachePathKey]uint64
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
		views:    make(map[netFileBatchCacheViewKey]uint64),
		paths:    make(map[netFileBatchCachePathKey]uint64),
		trees:    make(map[netFileBatchCachePathKey]uint64),
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

	c.setLocked(key, batch, now)
}

func (c *netFileBatchCache) SetIfGeneration(key netFileBatchCacheKey, batch netFileBatch, now time.Time, generation uint64) bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.generationLocked(key) != generation {
		return false
	}
	c.setLocked(key, batch, now)
	return true
}

func (c *netFileBatchCache) Generation(key netFileBatchCacheKey) uint64 {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.generationLocked(key)
}

func (c *netFileBatchCache) setLocked(key netFileBatchCacheKey, batch netFileBatch, now time.Time) {
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

	viewKey := netFileBatchCacheViewKey{
		SourceType: sourceType,
		AccountID:  accountID,
		Path:       path,
		SortBy:     sortBy,
		SortOrder:  sortOrder,
		Filter:     filter,
	}
	c.views[viewKey]++
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

	c.paths[netFileBatchCachePathKey{
		SourceType: sourceType,
		AccountID:  accountID,
		Path:       path,
	}]++
	for key := range c.items {
		if key.SourceType == sourceType && key.AccountID == accountID && key.Path == path {
			c.deleteLocked(key)
		}
	}
}

func (c *netFileBatchCache) InvalidatePathTree(sourceType string, accountID uint, path string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.trees[netFileBatchCachePathKey{
		SourceType: sourceType,
		AccountID:  accountID,
		Path:       path,
	}]++
	for key := range c.items {
		if key.SourceType == sourceType &&
			key.AccountID == accountID &&
			netFileCachePathInTree(key.Path, path) {
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

func (c *netFileBatchCache) generationLocked(key netFileBatchCacheKey) uint64 {
	generation := c.views[key.viewKey()] + c.paths[key.pathKey()]
	for treeKey, treeGeneration := range c.trees {
		if treeKey.SourceType == key.SourceType &&
			treeKey.AccountID == key.AccountID &&
			netFileCachePathInTree(key.Path, treeKey.Path) {
			generation += treeGeneration
		}
	}
	return generation
}

func (k netFileBatchCacheKey) viewKey() netFileBatchCacheViewKey {
	return netFileBatchCacheViewKey{
		SourceType: k.SourceType,
		AccountID:  k.AccountID,
		Path:       k.Path,
		SortBy:     k.SortBy,
		SortOrder:  k.SortOrder,
		Filter:     k.Filter,
	}
}

func (k netFileBatchCacheKey) pathKey() netFileBatchCachePathKey {
	return netFileBatchCachePathKey{
		SourceType: k.SourceType,
		AccountID:  k.AccountID,
		Path:       k.Path,
	}
}

func netFileCachePathInTree(path string, root string) bool {
	if root == "/" {
		return path == "/" || strings.HasPrefix(path, "/")
	}
	return path == root || strings.HasPrefix(path, root+"/")
}
