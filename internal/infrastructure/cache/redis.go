package cache

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisCache struct {
	client *redis.Client
}

func NewRedisCache(address string, port int, password string, db int) (*RedisCache, error) {
	addr := fmt.Sprintf("%s:%s", address, strconv.Itoa(port))
	c := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	if _, err := c.Ping(ctx).Result(); err != nil {
		return nil, fmt.Errorf("redis cache: ping: %w", err)
	}
	return &RedisCache{client: c}, nil
}

func (r *RedisCache) Get(ctx context.Context, key string) (*CacheValue, error) {
	v, err := r.client.Get(ctx, key).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, fmt.Errorf("redis cache: key %q not found", key)
		}
		return nil, fmt.Errorf("redis cache: get %q: %w", key, err)
	}
	return NewCacheValue(v)
}

func (r *RedisCache) Set(ctx context.Context, key string, value any, expiresAfter int64) error {
	expires := time.Duration(expiresAfter * int64(time.Second))
	if err := r.client.Set(ctx, key, value, expires).Err(); err != nil {
		return fmt.Errorf("redis cache: set %q: %w", key, err)
	}
	return nil
}

func (r *RedisCache) Delete(ctx context.Context, key string) error {
	if err := r.client.Del(ctx, key).Err(); err != nil {
		return fmt.Errorf("redis cache: delete %q: %w", key, err)
	}
	return nil
}
