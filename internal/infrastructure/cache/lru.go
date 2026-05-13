package cache

// moved from model/lru.go — see model/lru.go for re-export aliases

import (
	"container/list"
	"sync"
	"time"
)

// ObjectCache is the caching interface used across the application.
type ObjectCache interface {
	AddWithExpiresInSecs(key, value interface{}, expireAtSecs int64)
	AddWithDefaultExpires(key, value interface{})
	Add(key, value interface{})
	Purge()
	Get(key interface{}) (value interface{}, ok bool)
	Remove(key interface{})
	Len() int
	Name() string
	GetInvalidateClusterEvent() string
}

// LRUCache is a thread-safe fixed size LRU cache.
type LRUCache struct {
	size                   int
	evictList              *list.List
	items                  map[interface{}]*list.Element
	lock                   sync.RWMutex
	name                   string
	defaultExpiry          int64
	invalidateClusterEvent string
	currentGeneration      int64
	len                    int
}

// lruEntry is used to hold a value in the evictList.
type lruEntry struct {
	key          interface{}
	value        interface{}
	expireAtSecs int64
	generation   int64
}

// NewLru creates an LRU of the given size.
func NewLru(size int) *LRUCache {
	return &LRUCache{
		size:      size,
		evictList: list.New(),
		items:     make(map[interface{}]*list.Element, size),
	}
}

// NewLruWithParams creates an LRU with named parameters.
func NewLruWithParams(size int, name string, defaultExpiry int64, invalidateClusterEvent string) *LRUCache {
	lru := NewLru(size)
	lru.name = name
	lru.defaultExpiry = defaultExpiry
	lru.invalidateClusterEvent = invalidateClusterEvent
	return lru
}

// Purge is used to completely clear the cache.
func (c *LRUCache) Purge() {
	c.lock.Lock()
	defer c.lock.Unlock()

	c.len = 0
	c.currentGeneration++
}

func (c *LRUCache) Add(key, value interface{}) {
	c.AddWithExpiresInSecs(key, value, 0)
}

func (c *LRUCache) AddWithDefaultExpires(key, value interface{}) {
	c.AddWithExpiresInSecs(key, value, c.defaultExpiry)
}

func (c *LRUCache) AddWithExpiresInSecs(key, value interface{}, expireAtSecs int64) {
	c.lock.Lock()
	defer c.lock.Unlock()

	if expireAtSecs > 0 {
		expireAtSecs = (time.Now().UnixNano() / int64(time.Second)) + expireAtSecs
	}

	// Check for existing item
	if ent, ok := c.items[key]; ok {
		c.evictList.MoveToFront(ent)
		e := ent.Value.(*lruEntry)
		e.value = value
		e.expireAtSecs = expireAtSecs
		if e.generation != c.currentGeneration {
			e.generation = c.currentGeneration
			c.len++
		}
		return
	}

	// Add new item
	ent := &lruEntry{key, value, expireAtSecs, c.currentGeneration}
	elem := c.evictList.PushFront(ent)
	c.items[key] = elem
	c.len++

	if c.evictList.Len() > c.size {
		c.removeElement(c.evictList.Back())
	}
}

func (c *LRUCache) Get(key interface{}) (value interface{}, ok bool) {
	c.lock.Lock()
	defer c.lock.Unlock()

	if ent, ok := c.items[key]; ok {
		e := ent.Value.(*lruEntry)

		if e.generation != c.currentGeneration || (e.expireAtSecs > 0 && (time.Now().UnixNano()/int64(time.Second)) > e.expireAtSecs) {
			c.removeElement(ent)
			return nil, false
		}

		c.evictList.MoveToFront(ent)
		return ent.Value.(*lruEntry).value, true
	}

	return nil, false
}

func (c *LRUCache) Remove(key interface{}) {
	c.lock.Lock()
	defer c.lock.Unlock()

	if ent, ok := c.items[key]; ok {
		c.removeElement(ent)
	}
}

// Keys returns a slice of the keys in the cache, from oldest to newest.
func (c *LRUCache) Keys() []interface{} {
	c.lock.RLock()
	defer c.lock.RUnlock()

	keys := make([]interface{}, c.len)
	i := 0
	for ent := c.evictList.Back(); ent != nil; ent = ent.Prev() {
		e := ent.Value.(*lruEntry)
		if e.generation == c.currentGeneration {
			keys[i] = e.key
			i++
		}
	}

	return keys
}

// Len returns the number of items in the cache.
func (c *LRUCache) Len() int {
	c.lock.RLock()
	defer c.lock.RUnlock()
	return c.len
}

func (c *LRUCache) Name() string {
	return c.name
}

func (c *LRUCache) GetInvalidateClusterEvent() string {
	return c.invalidateClusterEvent
}

// removeElement is used to remove a given list element from the cache.
func (c *LRUCache) removeElement(e *list.Element) {
	c.evictList.Remove(e)
	kv := e.Value.(*lruEntry)
	if kv.generation == c.currentGeneration {
		c.len--
	}
	delete(c.items, kv.key)
}
