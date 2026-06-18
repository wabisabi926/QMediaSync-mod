package db

import (
	"Q115-STRM/internal/helpers"

	"github.com/coocood/freecache"
)

type CacheGlobal struct {
	CacheInstance *freecache.Cache
	CacheSize     int
}

var Cache CacheGlobal
var DefaultExpire = 300 // 默认5分钟过期

// 初始化缓存
func InitCache() {
	cacheSize := helpers.GlobalConfig.CacheSize
	Cache = CacheGlobal{
		CacheInstance: freecache.NewCache(cacheSize),
		CacheSize:     cacheSize,
	}
	helpers.AppLogger.Info("成功初始化内存缓存组件")
}

// expire设置为-1则代表取默认值
func (c *CacheGlobal) Set(key string, value []byte, expire int) {
	keyHash := helpers.MD5Hash(key)
	keyBytes := []byte(keyHash)
	if expire == -1 {
		expire = DefaultExpire
	}
	c.CacheInstance.Set(keyBytes, value, expire)
}

func (c *CacheGlobal) Get(key string) []byte {
	keyHash := helpers.MD5Hash(key)
	keyBytes := []byte(keyHash)
	value, err := c.CacheInstance.Get(keyBytes)
	if err != nil {
		return nil
	}
	return value
}

func (c *CacheGlobal) Delete(key string) {
	keyHash := helpers.MD5Hash(key)
	keyBytes := []byte(keyHash)
	c.CacheInstance.Del(keyBytes)
}
