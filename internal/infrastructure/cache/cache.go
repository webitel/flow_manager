package cache

import (
	"context"
	"errors"
)

type CacheType string

const (
	Memory CacheType = "memory"
	Redis  CacheType = "redis"
)

// CacheStores is a keyed map of available cache backends; used as an fx-injectable named type.
type CacheStores map[CacheType]CacheStore

type CacheValue struct {
	value any
}

type CacheStore interface {
	Get(ctx context.Context, key string) (*CacheValue, error)
	Set(ctx context.Context, key string, value any, expiresAfter int64) error
	Delete(ctx context.Context, key string) error
}

func (v *CacheValue) String() (string, error) {
	if v.value == nil {
		return "", errors.New("cache value is nil")
	}
	if s, ok := v.value.(string); ok {
		return s, nil
	}
	return "", errors.New("cache value: unable to convert to string")
}

func (v *CacheValue) Raw() any {
	return v
}

func (v *CacheValue) Set(value any) error {
	if value == nil {
		return errors.New("cache value: accepted value is nil")
	}
	v.value = value
	return nil
}

func NewCacheValue(value any) (*CacheValue, error) {
	var cv CacheValue
	if err := cv.Set(value); err != nil {
		return nil, err
	}
	return &cv, nil
}
