package model

import (
	"maps"
	"sync"
)

type ThreadSafeStringMap struct {
	sync.RWMutex
	internal map[string]string
}

func NewThreadSafeStringMap() *ThreadSafeStringMap {
	return &ThreadSafeStringMap{
		internal: make(map[string]string),
	}
}

func (sm *ThreadSafeStringMap) Load(key string) (string, bool) {
	sm.RLock()
	result, ok := sm.internal[key]
	sm.RUnlock()
	return result, ok
}

func (sm *ThreadSafeStringMap) Delete(key string) {
	sm.Lock()
	delete(sm.internal, key)
	sm.Unlock()
}

func (sm *ThreadSafeStringMap) Store(key, value string) {
	sm.Lock()
	sm.internal[key] = value
	sm.Unlock()
}

func (sm *ThreadSafeStringMap) Data() map[string]string {
	sm.RLock()
	defer sm.RUnlock()

	return maps.Clone(sm.internal)
}

func (sm *ThreadSafeStringMap) UnionMap(m map[string]string) {
	sm.Lock()
	maps.Copy(sm.internal, m)
	sm.Unlock()
}
