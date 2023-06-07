package app

import (
	"context"
	"net/http"

	"github.com/webitel/flow_manager/model"
	"github.com/webitel/flow_manager/store/cachelayer"
	"golang.org/x/sync/singleflight"
)

var g = singleflight.Group{}

func (fm *FlowManager) CacheSetValue(ctx context.Context, cacheType string, domainId int64, key string, value string, expiresAfter int64) *model.AppError {
	_, cacheKey := cachelayer.FormatKeys(cacheType, "set", domainId, key)
	return fm.cacheSetValue(ctx, cacheType, cacheKey, value, expiresAfter)
}

func (fm *FlowManager) cacheSetValue(ctx context.Context, cacheType string, key string, value string, expiresAfter int64) *model.AppError {
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
	workerKey, cacheKey := cachelayer.FormatKeys(cacheType, "get", domainId, key)
	v, appErr, _ := g.Do(workerKey, func() (interface{}, error) {
		value, err := fm.cacheGetValue(ctx, cacheType, cacheKey)
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
func (fm *FlowManager) cacheGetValue(ctx context.Context, cacheType string, key string) (*cachelayer.CacheValue, *model.AppError) {
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
	workerKey, cacheKey := cachelayer.FormatKeys(cacheType, "delete", domainId, key)
	_, err, _ := g.Do(workerKey, func() (interface{}, error) {
		return nil, fm.cacheDeleteValue(ctx, cacheType, cacheKey)
	})

	if err != nil {
		return err.(*model.AppError)
	}

	return nil
}
func (fm *FlowManager) cacheDeleteValue(ctx context.Context, cacheType string, key string) *model.AppError {
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

func (fm *FlowManager) GetCacheStoreByType(cacheType string) (cachelayer.CacheStore, *model.AppError) {
	v, ok := fm.cacheStore[cacheType]
	if !ok {
		return nil, model.NewAppError("App", "app.flow.get_cache_store", nil, "no such cache type", http.StatusBadRequest)
	}

	return v, nil
}
