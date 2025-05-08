package cachelayer

import (
	"context"
	"fmt"
	"net/http"

	"github.com/webitel/flow_manager/model"
)

type MemoryCache struct {
	lruCache model.ObjectCache
}

type MemoryCacheConfig struct {
	Size          int
	DefaultExpiry int64
}

func NewMemoryCache(conf *MemoryCacheConfig) *MemoryCache {
	return &MemoryCache{lruCache: model.NewLruWithParams(conf.Size, "memoryCache", int64(conf.DefaultExpiry), "")}
}

func (m *MemoryCache) Get(ctx context.Context, key string) (*CacheValue, *model.AppError) {
	value, ok := m.lruCache.Get(key)
	if !ok {
		return nil, model.NewAppError("CacheLayer.MemoryCache", "cache.memory_cache.get", nil, fmt.Sprintf("unable to find value by key %s", key), http.StatusInternalServerError)
	}
	return NewCacheValue(value)
}

func (m *MemoryCache) Set(ctx context.Context, key string, value any, expiresAfterSecs int64) *model.AppError {
	m.lruCache.AddWithExpiresInSecs(key, value, expiresAfterSecs)
	return nil
}
func (m *MemoryCache) Delete(ctx context.Context, key string) *model.AppError {
	m.lruCache.Remove(key)
	return nil
}

func (m *MemoryCache) IsValid() *model.AppError {
	if m.lruCache == nil {
		return model.NewAppError("CacheLayer.MemoryCache", "cache.memory_cache", nil, "lru cache client is not declared", http.StatusInternalServerError)
	}
	return nil
}
