package cachelayer

import (
	"context"
	"net/http"

	"github.com/webitel/flow_manager/model"
)

type CacheValue struct {
	value any
}

type CacheStore interface {
	Get(ctx context.Context, key string) (*CacheValue, *model.AppError)
	Set(ctx context.Context, key string, value any, expiresAfter int64) *model.AppError
	Delete(ctx context.Context, key string) *model.AppError
}

func (v *CacheValue) String() (string, *model.AppError) {
	if v.value == nil {
		return "", model.NewAppError("CacheValue", "cache_value.string", nil, "value is nil", http.StatusInternalServerError)
	}
	if v, ok := v.value.(string); ok {
		return v, nil
	} else {
		return "", model.NewAppError("CacheValue", "cache_value.string", nil, "unable to convert value", http.StatusInternalServerError)
	}

}

func (v *CacheValue) Raw() any {
	return v
}
func (v *CacheValue) Set(value any) *model.AppError {
	if value != nil {
		v.value = value
		return nil
	} else {
		return model.NewAppError("CacheValue", "cache_value.set", nil, "accepted value is nil", http.StatusInternalServerError)
	}
}

func NewCacheValue(value any) (*CacheValue, *model.AppError) {
	var cv CacheValue
	err := cv.Set(value)
	if err != nil {
		return nil, err
	}
	return &cv, err
}
