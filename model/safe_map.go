package model

import "github.com/webitel/flow_manager/internal/infrastructure/cache"

// Re-export for backward compatibility.
type ThreadSafeStringMap = cache.ThreadSafeStringMap

// NewThreadSafeStringMap creates a new ThreadSafeStringMap.
func NewThreadSafeStringMap() *ThreadSafeStringMap {
	return cache.NewThreadSafeStringMap()
}
