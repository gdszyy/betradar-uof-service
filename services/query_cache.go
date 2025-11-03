package services

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

// QueryCache 查询结果缓存
type QueryCache struct {
	cache map[string]*CacheEntry
	mu    sync.RWMutex
	ttl   time.Duration
}

// CacheEntry 缓存条目
type CacheEntry struct {
	Data      interface{}
	ExpiresAt time.Time
}

// NewQueryCache 创建查询缓存
func NewQueryCache(ttl time.Duration) *QueryCache {
	cache := &QueryCache{
		cache: make(map[string]*CacheEntry),
		ttl:   ttl,
	}
	
	// 启动清理协程
	go cache.cleanupLoop()
	
	return cache
}

// Get 获取缓存
func (c *QueryCache) Get(key string) (interface{}, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	entry, exists := c.cache[key]
	if !exists {
		return nil, false
	}
	
	// 检查是否过期
	if time.Now().After(entry.ExpiresAt) {
		return nil, false
	}
	
	return entry.Data, true
}

// Set 设置缓存
func (c *QueryCache) Set(key string, data interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	c.cache[key] = &CacheEntry{
		Data:      data,
		ExpiresAt: time.Now().Add(c.ttl),
	}
}

// Delete 删除缓存
func (c *QueryCache) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	delete(c.cache, key)
}

// Clear 清空缓存
func (c *QueryCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	c.cache = make(map[string]*CacheEntry)
}

// Size 获取缓存大小
func (c *QueryCache) Size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	return len(c.cache)
}

// cleanupLoop 定期清理过期缓存
func (c *QueryCache) cleanupLoop() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	
	for range ticker.C {
		c.cleanup()
	}
}

// cleanup 清理过期缓存
func (c *QueryCache) cleanup() {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	now := time.Now()
	for key, entry := range c.cache {
		if now.After(entry.ExpiresAt) {
			delete(c.cache, key)
		}
	}
}

// GenerateCacheKey 生成缓存键
func GenerateCacheKey(prefix string, params interface{}) string {
	// 将参数序列化为 JSON
	jsonBytes, err := json.Marshal(params)
	if err != nil {
		// 如果序列化失败,使用时间戳作为键(不缓存)
		return fmt.Sprintf("%s_%d", prefix, time.Now().UnixNano())
	}
	
	// 使用 SHA256 生成哈希
	hash := sha256.Sum256(jsonBytes)
	return fmt.Sprintf("%s_%x", prefix, hash[:16]) // 使用前 16 字节
}

