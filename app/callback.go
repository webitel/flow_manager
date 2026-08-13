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
	cb           *expirable.LRU[string, CbFunc]
	dialogByCall *expirable.LRU[string, string] // callId -> dialogId, bound for the bot session lifetime
}

func NewCallbackResolver() *CallbackResolver {
	return &CallbackResolver{
		cb:           expirable.NewLRU[string, CbFunc](callbackSize, nil, callbackExpire),
		dialogByCall: expirable.NewLRU[string, string](callbackSize, nil, callbackExpire),
	}
}

func (c *CallbackResolver) Register(id string, fn CbFunc) {
	c.cb.Add(id, fn)
}

// BindDialog indexes the ai_bots dialog id by the flow call id (conn.Id()) so
// applications running inside the live bot session (e.g. embed) can resolve it.
func (c *CallbackResolver) BindDialog(callId, dialogId string) {
	c.dialogByCall.Add(callId, dialogId)
}

// UnbindDialog drops the call -> dialog mapping when the bot session ends.
func (c *CallbackResolver) UnbindDialog(callId string) {
	c.dialogByCall.Remove(callId)
}

// DialogByCall resolves the dialog id bound to a call id; ok=false if none is live.
func (c *CallbackResolver) DialogByCall(callId string) (string, bool) {
	return c.dialogByCall.Get(callId)
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
