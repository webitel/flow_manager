package app

import (
	"context"
	"fmt"
	"strconv"

	"github.com/webitel/flow_manager/internal/infrastructure/cache"
	"golang.org/x/sync/singleflight"
)

var g = singleflight.Group{}

func (fm *FlowManager) CacheSetValue(ctx context.Context, cacheType string, domainId int64, key string, value any, expiresAfter int64) error {
	cacheTypeParsed := parseCacheType(cacheType)
	_, cacheKey := formatKeys(cacheTypeParsed, "set", domainId, key)
	return fm.cacheSetValue(ctx, cacheTypeParsed, cacheKey, value, expiresAfter)
}

func (fm *FlowManager) cacheSetValue(ctx context.Context, cacheType cache.CacheType, key string, value any, expiresAfter int64) error {
	v, err := fm.GetCacheStoreByType(cacheType)
	if err != nil {
		return err
	}
	return v.Set(ctx, key, value, expiresAfter)
}

func (fm *FlowManager) CacheGetValue(ctx context.Context, cacheType string, domainId int64, key string) (*cache.CacheValue, error) {
	cacheTypeParsed := parseCacheType(cacheType)
	workerKey, cacheKey := formatKeys(cacheTypeParsed, "get", domainId, key)
	v, err, _ := g.Do(workerKey, func() (interface{}, error) {
		return fm.cacheGetValue(ctx, cacheTypeParsed, cacheKey)
	})
	if err != nil {
		return nil, err
	}
	return v.(*cache.CacheValue), nil
}

func (fm *FlowManager) cacheGetValue(ctx context.Context, cacheType cache.CacheType, key string) (*cache.CacheValue, error) {
	v, err := fm.GetCacheStoreByType(cacheType)
	if err != nil {
		return nil, err
	}
	return v.Get(ctx, key)
}

func (fm *FlowManager) CacheDeleteValue(ctx context.Context, cacheType string, domainId int64, key string) error {
	cacheTypeParsed := parseCacheType(cacheType)
	workerKey, cacheKey := formatKeys(cacheTypeParsed, "delete", domainId, key)
	_, err, _ := g.Do(workerKey, func() (interface{}, error) {
		return nil, fm.cacheDeleteValue(ctx, cacheTypeParsed, cacheKey)
	})
	return err
}

func (fm *FlowManager) cacheDeleteValue(ctx context.Context, cacheType cache.CacheType, key string) error {
	v, err := fm.GetCacheStoreByType(cacheType)
	if err != nil {
		return err
	}
	return v.Delete(ctx, key)
}

func (fm *FlowManager) GetCacheStoreByType(cacheType cache.CacheType) (cache.CacheStore, error) {
	v, ok := fm.cacheStore[cacheType]
	if !ok {
		fm.log.Debug(fmt.Sprintf("unable to find given cache storage (%s), setting memory storage..", cacheType))
		return fm.cacheStore[cache.Memory], nil
	}
	return v, nil
}

func parseCacheType(cacheType string) cache.CacheType {
	switch cacheType {
	case string(cache.Redis):
		return cache.Redis
	default:
		return cache.Memory
	}
}

func (fm *FlowManager) GetCookieCache(ctx context.Context, domainID int64, key string) (string, error) {
	v, err := fm.CacheGetValue(ctx, string(cache.Memory), domainID, key)
	if err != nil {
		return "", err
	}
	return v.String()
}

func (fm *FlowManager) SetCookieCache(ctx context.Context, domainID int64, key string, value string, ttlSecs int64) error {
	return fm.CacheSetValue(ctx, string(cache.Memory), domainID, key, value, ttlSecs)
}

func formatKeys(cacheType cache.CacheType, method string, domainId int64, key string) (workerKey string, cacheKey string) {
	cacheKey = fmt.Sprintf("%s.%s", strconv.FormatInt(domainId, 10), key)
	workerKey = fmt.Sprintf("%s.%s.%s", cacheType, method, cacheKey)
	return
}
