package cachelayer

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/webitel/flow_manager/model"
)

type RedisCache struct {
	redis *redis.Client
}

func NewRedisCache(address string, port int, password string, db int) (*RedisCache, *model.AppError) {
	var redisCache RedisCache
	address = fmt.Sprintf("%s:%s", address, strconv.Itoa(port))
	redisCache.redis = redis.NewClient(&redis.Options{
		Addr:     address,
		Password: password,
		DB:       db,
	})
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	_, err := redisCache.redis.Ping(ctx).Result()
	if err != nil {
		return nil, model.NewAppError("CacheLayer.RedisCache", "cache.redis_cache", nil, err.Error(), http.StatusInternalServerError)
	}
	return &redisCache, nil
}

func (r *RedisCache) Get(ctx context.Context, key string) (*CacheValue, *model.AppError) {
	if err := r.IsValid(); err != nil {
		return nil, err
	}
	v, err := r.redis.Get(ctx, key).Result()
	if err != nil || err == redis.Nil {
		return nil, model.NewAppError("CacheLayer.RedisCache", "cache.redis_cache.get", nil, err.Error(), http.StatusInternalServerError)
	}
	return NewCacheValue(v)
}

func (r *RedisCache) Set(ctx context.Context, key string, value any, expiresAfter int64) *model.AppError {
	if err := r.IsValid(); err != nil {
		return err
	}
	expires := time.Duration(expiresAfter * int64(time.Second))
	err := r.redis.Set(ctx, key, value, expires).Err()
	if err != nil {
		return model.NewAppError("CacheLayer.RedisCache", "cache.redis_cache.set", nil, err.Error(), http.StatusInternalServerError)
	}
	return nil
}

func (r *RedisCache) Delete(ctx context.Context, key string) *model.AppError {
	if err := r.IsValid(); err != nil {
		return err
	}
	err := r.redis.Del(ctx, key).Err()
	if err != nil {
		return model.NewAppError("CacheLayer.RedisCache", "cache.redis_cache.delete", nil, err.Error(), http.StatusInternalServerError)
	}
	return nil
}

func (r *RedisCache) IsValid() *model.AppError {
	if r.redis == nil {
		return model.NewAppError("CacheLayer.RedisCache", "cache.redis_cache", nil, "redis client not declared", http.StatusInternalServerError)
	}
	return nil
}
