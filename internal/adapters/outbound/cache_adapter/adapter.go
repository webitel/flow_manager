// Package cache_adapter wraps cache.CacheStores and exposes cache get/set/delete
// methods that used to live in app/cache.go.
package cache_adapter

import (
	"context"
	"fmt"
	"strconv"

	"github.com/webitel/wlog"

	"github.com/webitel/flow_manager/internal/infrastructure/cache"
)

// CacheAdapter wraps CacheStores and exposes thin cache helpers.
// Embed *CacheAdapter in FlowManager to promote all methods without re-declaring
// them in app/.
type CacheAdapter struct {
	stores cache.CacheStores
	log    *wlog.Logger
}

var g = singleflightGroup{}

// New creates a new Adapter.
func New(stores cache.CacheStores, log *wlog.Logger) *CacheAdapter {
	return &CacheAdapter{stores: stores, log: log}
}

func (a *CacheAdapter) GetCacheStoreByType(cacheType cache.CacheType) (cache.CacheStore, error) {
	v, ok := a.stores[cacheType]
	if !ok {
		a.log.Debug(fmt.Sprintf("unable to find given cache storage (%s), setting memory storage..", cacheType))
		return a.stores[cache.Memory], nil
	}
	return v, nil
}

func (a *CacheAdapter) CacheSetValue(ctx context.Context, cacheType string, domainId int64, key string, value any, expiresAfter int64) error {
	ct := parseCacheType(cacheType)
	_, cacheKey := formatKeys(ct, "set", domainId, key)
	return a.cacheSetValue(ctx, ct, cacheKey, value, expiresAfter)
}

func (a *CacheAdapter) cacheSetValue(ctx context.Context, cacheType cache.CacheType, key string, value any, expiresAfter int64) error {
	v, err := a.GetCacheStoreByType(cacheType)
	if err != nil {
		return err
	}
	return v.Set(ctx, key, value, expiresAfter)
}

func (a *CacheAdapter) CacheGetValue(ctx context.Context, cacheType string, domainId int64, key string) (*cache.CacheValue, error) {
	ct := parseCacheType(cacheType)
	workerKey, cacheKey := formatKeys(ct, "get", domainId, key)
	v, err, _ := g.Do(workerKey, func() (interface{}, error) {
		return a.cacheGetValue(ctx, ct, cacheKey)
	})
	if err != nil {
		return nil, err
	}
	return v.(*cache.CacheValue), nil
}

func (a *CacheAdapter) cacheGetValue(ctx context.Context, cacheType cache.CacheType, key string) (*cache.CacheValue, error) {
	v, err := a.GetCacheStoreByType(cacheType)
	if err != nil {
		return nil, err
	}
	return v.Get(ctx, key)
}

func (a *CacheAdapter) CacheDeleteValue(ctx context.Context, cacheType string, domainId int64, key string) error {
	ct := parseCacheType(cacheType)
	workerKey, cacheKey := formatKeys(ct, "delete", domainId, key)
	_, err, _ := g.Do(workerKey, func() (interface{}, error) {
		return nil, a.cacheDeleteValue(ctx, ct, cacheKey)
	})
	return err
}

func (a *CacheAdapter) cacheDeleteValue(ctx context.Context, cacheType cache.CacheType, key string) error {
	v, err := a.GetCacheStoreByType(cacheType)
	if err != nil {
		return err
	}
	return v.Delete(ctx, key)
}

func (a *CacheAdapter) GetCookieCache(ctx context.Context, domainID int64, key string) (string, error) {
	v, err := a.CacheGetValue(ctx, string(cache.Memory), domainID, key)
	if err != nil {
		return "", err
	}
	return v.String()
}

func (a *CacheAdapter) SetCookieCache(ctx context.Context, domainID int64, key string, value string, ttlSecs int64) error {
	return a.CacheSetValue(ctx, string(cache.Memory), domainID, key, value, ttlSecs)
}

func (a *CacheAdapter) CacheGet(ctx context.Context, cacheType string, domainID int64, key string) (string, error) {
	v, err := a.CacheGetValue(ctx, cacheType, domainID, key)
	if err != nil {
		return "", err
	}
	return v.String()
}

func (a *CacheAdapter) CacheSet(ctx context.Context, cacheType string, domainID int64, key string, value string, ttlSecs int64) error {
	return a.CacheSetValue(ctx, cacheType, domainID, key, value, ttlSecs)
}

func (a *CacheAdapter) CacheDelete(ctx context.Context, cacheType string, domainID int64, key string) error {
	return a.CacheDeleteValue(ctx, cacheType, domainID, key)
}

func parseCacheType(cacheType string) cache.CacheType {
	switch cacheType {
	case string(cache.Redis):
		return cache.Redis
	default:
		return cache.Memory
	}
}

func formatKeys(cacheType cache.CacheType, method string, domainId int64, key string) (workerKey string, cacheKey string) {
	cacheKey = fmt.Sprintf("%s.%s", strconv.FormatInt(domainId, 10), key)
	workerKey = fmt.Sprintf("%s.%s.%s", cacheType, method, cacheKey)
	return
}
