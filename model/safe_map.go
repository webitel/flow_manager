package model

import "sync"

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
	sm.Lock()
	data := sm.internal
	sm.Unlock()
	return data
}

func (sm *ThreadSafeStringMap) UnionMap(m map[string]string) {
	sm.Lock()
	for k, v := range m {
		sm.internal[k] = v
	}
	sm.Unlock()
}
