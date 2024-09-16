package app

import (
	"context"
	"fmt"
	"strconv"

	"github.com/webitel/flow_manager/model"
	"github.com/webitel/flow_manager/store/cachelayer"
	"golang.org/x/sync/singleflight"
)

var g = singleflight.Group{}

type CacheType string

const (
	Memory CacheType = "memory"
	Redis  CacheType = "redis"
)

// Cache set value sets value to the given type of cache storage. (expiresAfter in seconds!)
func (fm *FlowManager) CacheSetValue(ctx context.Context, cacheType string, domainId int64, key string, value any, expiresAfter int64) *model.AppError {
	cacheTypeParsed := parseCacheType(cacheType)
	_, cacheKey := formatKeys(cacheTypeParsed, "set", domainId, key)
	return fm.cacheSetValue(ctx, cacheTypeParsed, cacheKey, value, expiresAfter)
}

func (fm *FlowManager) cacheSetValue(ctx context.Context, cacheType CacheType, key string, value any, expiresAfter int64) *model.AppError {
	v, appErr := fm.GetCacheStoreByType(cacheType)
	if appErr != nil {
		return appErr
	}

	err := v.Set(ctx, key, value, expiresAfter)
	if err != nil {
		return err
	}
	return nil
}
func (fm *FlowManager) CacheGetValue(ctx context.Context, cacheType string, domainId int64, key string) (*cachelayer.CacheValue, *model.AppError) {
	cacheTypeParsed := parseCacheType(cacheType)
	workerKey, cacheKey := formatKeys(cacheTypeParsed, "get", domainId, key)
	v, appErr, _ := g.Do(workerKey, func() (interface{}, error) {
		value, err := fm.cacheGetValue(ctx, cacheTypeParsed, cacheKey)
		if err != nil {
			return nil, err
		}
		return value, nil
	})

	if appErr != nil {
		return nil, appErr.(*model.AppError)
	}

	return v.(*cachelayer.CacheValue), nil
}
func (fm *FlowManager) cacheGetValue(ctx context.Context, cacheType CacheType, key string) (*cachelayer.CacheValue, *model.AppError) {
	v, err := fm.GetCacheStoreByType(cacheType)
	if err != nil {
		return nil, err
	}

	value, appErr := v.Get(ctx, key)
	if appErr != nil {
		return nil, appErr
	}
	return value, nil
}

func (fm *FlowManager) CacheDeleteValue(ctx context.Context, cacheType string, domainId int64, key string) *model.AppError {
	cacheTypeParsed := parseCacheType(cacheType)
	workerKey, cacheKey := formatKeys(cacheTypeParsed, "delete", domainId, key)
	_, err, _ := g.Do(workerKey, func() (interface{}, error) {
		return nil, fm.cacheDeleteValue(ctx, cacheTypeParsed, cacheKey)
	})

	if err != nil {
		return err.(*model.AppError)
	}

	return nil
}
func (fm *FlowManager) cacheDeleteValue(ctx context.Context, cacheType CacheType, key string) *model.AppError {
	v, err := fm.GetCacheStoreByType(cacheType)
	if err != nil {
		return err
	}

	err = v.Delete(ctx, key)
	if err != nil {
		return err
	}
	return nil
}

func (fm *FlowManager) GetCacheStoreByType(cacheType CacheType) (cachelayer.CacheStore, *model.AppError) {
	v, ok := fm.cacheStore[cacheType]
	if !ok {
		fm.log.Debug(fmt.Sprintf("unable to find given cache storage (%s), setting memory storage..", cacheType))
		return fm.cacheStore["memory"], nil
	}

	return v, nil
}

func parseCacheType(cacheType string) CacheType {
	switch cacheType {
	case string(Redis):
		return Redis
	default:
		return Memory
	}
}

func formatKeys(cacheType CacheType, method string, domainId int64, key string) (workerKey string, cacheKey string) {
	cacheKey = fmt.Sprintf("%s.%s", strconv.FormatInt(domainId, 10), key)
	workerKey = fmt.Sprintf("%s.%s.%s", cacheType, method, cacheKey)
	return
}
