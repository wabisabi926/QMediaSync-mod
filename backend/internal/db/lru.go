package db

import (
	"container/list"
	"sync"
)

type LRUCache struct {
	capacity int
	cache    map[string]*list.Element
	list     *list.List
	mutex    sync.RWMutex
}

type entry struct {
	key   string
	value any
}

func NewLRUCache(capacity int) *LRUCache {
	return &LRUCache{
		capacity: capacity,
		cache:    make(map[string]*list.Element),
		list:     list.New(),
	}
}

func (lru *LRUCache) Get(key string) (any, bool) {
	lru.mutex.Lock()
	defer lru.mutex.Unlock()

	if elem, exists := lru.cache[key]; exists {
		lru.list.MoveToFront(elem)
		return elem.Value.(*entry).value, true
	}
	return nil, false
}

func (lru *LRUCache) Put(key string, value any) {
	lru.mutex.Lock()
	defer lru.mutex.Unlock()

	// 如果key已存在，更新值并移动到前面
	if elem, exists := lru.cache[key]; exists {
		lru.list.MoveToFront(elem)
		elem.Value.(*entry).value = value
		return
	}

	// 如果达到容量限制，移除最久未使用的元素
	if lru.list.Len() >= lru.capacity {
		tail := lru.list.Back()
		if tail != nil {
			lru.list.Remove(tail)
			delete(lru.cache, tail.Value.(*entry).key)
		}
	}

	// 添加新元素
	elem := lru.list.PushFront(&entry{key: key, value: value})
	lru.cache[key] = elem
}

func (lru *LRUCache) Len() int {
	lru.mutex.RLock()
	defer lru.mutex.RUnlock()
	return lru.list.Len()
}

func (lru *LRUCache) Clear() {
	lru.mutex.Lock()
	defer lru.mutex.Unlock()
	lru.cache = make(map[string]*list.Element)
	lru.list.Init()
}
