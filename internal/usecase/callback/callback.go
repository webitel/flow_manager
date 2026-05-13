package callback

import (
	"context"
	"errors"
	"time"

	"github.com/hashicorp/golang-lru/v2/expirable"
)

const (
	callbackExpire = time.Hour
	callbackSize   = 2000
)

// CbFunc is the function signature for registered callbacks.
type CbFunc func(ctx context.Context, data any) (any, error)

// Resolver manages short-lived in-memory callbacks identified by a string key.
type Resolver struct {
	cb *expirable.LRU[string, CbFunc]
}

// New creates a new Resolver.
func New() *Resolver {
	return &Resolver{
		cb: expirable.NewLRU[string, CbFunc](callbackSize, nil, callbackExpire),
	}
}

func (c *Resolver) Register(id string, fn CbFunc) {
	c.cb.Add(id, fn)
}

func (c *Resolver) Unregister(id string) error {
	ok := c.cb.Remove(id)
	if !ok {
		return errors.New("callback not found")
	}
	return nil
}

func (c *Resolver) Callback(ctx context.Context, id string, v any) (any, error) {
	cb, ok := c.cb.Get(id)
	if !ok {
		return nil, errors.New("callback not found")
	}
	return cb(ctx, v)
}
