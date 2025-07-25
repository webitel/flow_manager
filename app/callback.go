package app

import (
	"context"
	"errors"
	"github.com/hashicorp/golang-lru/v2/expirable"
	"time"
)

const (
	callbackExpire = time.Hour
	callbackSize   = 2000
)

type CbFunc func(ctx context.Context, data any) (any, error)

type CallbackResolver struct {
	cb *expirable.LRU[string, CbFunc]
}

func NewCallbackResolver() *CallbackResolver {
	return &CallbackResolver{
		cb: expirable.NewLRU[string, CbFunc](callbackSize, nil, callbackExpire),
	}
}

func (c *CallbackResolver) Register(id string, fn CbFunc) {
	c.cb.Add(id, fn)
}

func (c *CallbackResolver) Unregister(id string) error {
	ok := c.cb.Remove(id)
	if !ok {
		return errors.New("callback not found")
	}

	return nil
}

func (c *CallbackResolver) Callback(ctx context.Context, id string, v any) (any, error) {
	cb, ok := c.cb.Get(id)
	if !ok {
		return nil, errors.New("callback not found")
	}

	return cb(ctx, v)
}
