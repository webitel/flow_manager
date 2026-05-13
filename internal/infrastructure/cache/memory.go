package cache

import (
	"context"
	"fmt"
)

type MemoryCache struct {
	lruCache ObjectCache
}

type MemoryCacheConfig struct {
	Size          int
	DefaultExpiry int64
}

func NewMemoryCache(conf *MemoryCacheConfig) *MemoryCache {
	return &MemoryCache{lruCache: NewLruWithParams(conf.Size, "memoryCache", int64(conf.DefaultExpiry), "")}
}

func (m *MemoryCache) Get(_ context.Context, key string) (*CacheValue, error) {
	value, ok := m.lruCache.Get(key)
	if !ok {
		return nil, fmt.Errorf("memory cache: key %q not found", key)
	}
	return NewCacheValue(value)
}

func (m *MemoryCache) Set(_ context.Context, key string, value any, expiresAfterSecs int64) error {
	m.lruCache.AddWithExpiresInSecs(key, value, expiresAfterSecs)
	return nil
}

func (m *MemoryCache) Delete(_ context.Context, key string) error {
	m.lruCache.Remove(key)
	return nil
}
