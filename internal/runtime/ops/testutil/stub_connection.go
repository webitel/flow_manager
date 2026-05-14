// Package testutil provides shared test helpers for op unit tests.
// Import it only in _test.go files or test packages.
package testutil

import (
	"context"
	"sync"

	"github.com/webitel/wlog"

	"github.com/webitel/flow_manager/internal/domain/flow"
)

// StubResponse satisfies flow.Response for test assertions.
type StubResponse struct{ Msg string }

func (r StubResponse) String() string { return r.Msg }

var (
	ResponseOK  flow.Response = StubResponse{"OK"}
	ResponseErr flow.Response = StubResponse{"ERROR"}
)

// StubConnection is a hand-written, zero-dependency stub of flow.Connection.
// Configure its fields before use; all methods are safe to call without setup.
//
//	conn := &testutil.StubConnection{
//	    IDVal:       "test-conn-1",
//	    DomainIDVal: 42,
//	    Vars:        map[string]string{"lang": "uk"},
//	}
type StubConnection struct {
	IDVal       string
	DomainIDVal int64
	NodeIDVal   string
	TypeVal     flow.ConnectionType

	mu      sync.RWMutex
	Vars    map[string]string // readable via Get(); writeable via Set()
	SetVars []flow.Variables  // records all Set() calls for assertions

	// SetErr, when non-nil, is returned by Set().
	SetErr error
}

func (c *StubConnection) Type() flow.ConnectionType { return c.TypeVal }
func (c *StubConnection) Id() string                { return c.IDVal }
func (c *StubConnection) NodeId() string            { return c.NodeIDVal }
func (c *StubConnection) DomainId() int64           { return c.DomainIDVal }
func (c *StubConnection) Context() context.Context  { return context.Background() }
func (c *StubConnection) Close() error              { return nil }
func (c *StubConnection) Log() *wlog.Logger         { return wlog.GlobalLogger() }

func (c *StubConnection) Variables() map[string]string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	out := make(map[string]string, len(c.Vars))
	for k, v := range c.Vars {
		out[k] = v
	}
	return out
}

func (c *StubConnection) Get(key string) (string, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	v, ok := c.Vars[key]
	return v, ok
}

func (c *StubConnection) Set(_ context.Context, vars flow.Variables) (flow.Response, error) {
	if c.SetErr != nil {
		return ResponseErr, c.SetErr
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.Vars == nil {
		c.Vars = make(map[string]string)
	}
	c.SetVars = append(c.SetVars, vars)
	for k, v := range vars {
		c.Vars[k] = formatVar(v)
	}
	return ResponseOK, nil
}

func (c *StubConnection) ParseText(text string, _ ...flow.ParseOption) string {
	return text // no variable expansion in stub
}

func formatVar(v any) string {
	if v == nil {
		return ""
	}
	switch t := v.(type) {
	case string:
		return t
	default:
		return ""
	}
}

// SetCallCount returns how many times Set() was called.
func (c *StubConnection) SetCallCount() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.SetVars)
}
