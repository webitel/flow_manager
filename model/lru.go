package model

import "github.com/webitel/flow_manager/internal/infrastructure/cache"

// Re-export for backward compatibility.
type ObjectCache = cache.ObjectCache

// Cache is re-exported from internal/infrastructure/cache.
// Use cache.LRUCache directly in new code.
type Cache = cache.LRUCache

// NewLru creates an LRU of the given size.
func NewLru(size int) *Cache {
	return cache.NewLru(size)
}

// NewLruWithParams creates an LRU with named parameters.
func NewLruWithParams(size int, name string, defaultExpiry int64, invalidateClusterEvent string) *Cache {
	return cache.NewLruWithParams(size, name, defaultExpiry, invalidateClusterEvent)
}
