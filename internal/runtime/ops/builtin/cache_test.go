package builtin_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/webitel/flow_manager/internal/runtime/ops"
	"github.com/webitel/flow_manager/internal/runtime/ops/builtin"
	"github.com/webitel/flow_manager/internal/runtime/ops/testutil"
)

// ── fakeCacheDeps ──────────────────────────────────────────────────────────────

// fakeCacheDeps is a minimal hand-written fake for CacheDeps.
// It stores values in a plain map and records calls for assertions.
type fakeCacheDeps struct {
	store map[string]string // cacheType+":"+key → value
	err   error             // returned by all methods when non-nil
}

func newFakeCache() *fakeCacheDeps {
	return &fakeCacheDeps{store: map[string]string{}}
}

func cacheKey(cacheType, key string) string { return cacheType + ":" + key }

func (f *fakeCacheDeps) CacheGet(_ context.Context, cacheType string, _ int64, key string) (string, error) {
	if f.err != nil {
		return "", f.err
	}
	return f.store[cacheKey(cacheType, key)], nil
}

func (f *fakeCacheDeps) CacheSet(_ context.Context, cacheType string, _ int64, key, value string, _ int64) error {
	if f.err != nil {
		return f.err
	}
	f.store[cacheKey(cacheType, key)] = value
	return nil
}

func (f *fakeCacheDeps) CacheDelete(_ context.Context, cacheType string, _ int64, key string) error {
	if f.err != nil {
		return f.err
	}
	delete(f.store, cacheKey(cacheType, key))
	return nil
}

// ── helpers ────────────────────────────────────────────────────────────────────

func cacheInput(action, cacheType string, actionArgs map[string]any, vars map[string]string) ops.OpInput {
	args := map[string]any{"action": action, "type": cacheType, action: actionArgs}
	return testutil.MakeInputWithDomain(1, args, vars)
}

func runCache(deps *fakeCacheDeps, action, cacheType string, actionArgs map[string]any, vars map[string]string) (ops.OpOutput, error) {
	return builtin.CacheOp(deps).Execute(context.Background(), cacheInput(action, cacheType, actionArgs, vars))
}

// ── tests ──────────────────────────────────────────────────────────────────────

func TestCacheOp_Set(t *testing.T) {
	cases := []struct {
		name      string
		cacheType string
		key       string
		value     string
		ttl       string
		vars      map[string]string
		wantStored string
	}{
		{
			name:       "simple literal",
			cacheType:  "memory",
			key:        "greeting",
			value:      "hello",
			ttl:        "60",
			wantStored: "hello",
		},
		{
			name:       "variable expansion in value",
			cacheType:  "memory",
			key:        "msg",
			value:      "${name}",
			ttl:        "60",
			vars:       map[string]string{"name": "alice"},
			wantStored: "alice",
		},
		{
			name:       "redis cache type",
			cacheType:  "redis",
			key:        "token",
			value:      "abc123",
			ttl:        "3600",
			wantStored: "abc123",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			deps := newFakeCache()
			_, err := runCache(deps, "set", tc.cacheType,
				map[string]any{"data": map[string]any{tc.key: tc.value}, "ttl": tc.ttl},
				tc.vars,
			)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			got := deps.store[cacheKey(tc.cacheType, tc.key)]
			if got != tc.wantStored {
				t.Errorf("stored %q, want %q", got, tc.wantStored)
			}
		})
	}
}

func TestCacheOp_Get(t *testing.T) {
	cases := []struct {
		name       string
		seeded     map[string]string // pre-populated cache
		getMap     map[string]any    // varName → cacheKey
		wantSetVars map[string]string
	}{
		{
			name:       "single key",
			seeded:     map[string]string{"memory:session_id": "abc"},
			getMap:     map[string]any{"result": "session_id"},
			wantSetVars: map[string]string{"result": "abc"},
		},
		{
			name:       "missing key returns empty string",
			seeded:     map[string]string{},
			getMap:     map[string]any{"x": "nonexistent"},
			wantSetVars: map[string]string{"x": ""},
		},
		{
			name: "multiple keys",
			seeded: map[string]string{
				"memory:a": "1",
				"memory:b": "2",
			},
			getMap:     map[string]any{"va": "a", "vb": "b"},
			wantSetVars: map[string]string{"va": "1", "vb": "2"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			deps := &fakeCacheDeps{store: tc.seeded}
			out, err := runCache(deps, "get", "memory", tc.getMap, nil)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			for k, want := range tc.wantSetVars {
				if got := out.SetVars[k]; got != want {
					t.Errorf("SetVars[%q] = %q, want %q", k, got, want)
				}
			}
		})
	}
}

func TestCacheOp_Delete(t *testing.T) {
	deps := &fakeCacheDeps{store: map[string]string{
		"memory:k1": "v1",
		"memory:k2": "v2",
	}}
	_, err := runCache(deps, "delete", "memory",
		map[string]any{"keys": []any{"k1"}}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := deps.store["memory:k1"]; ok {
		t.Error("k1 should have been deleted")
	}
	if deps.store["memory:k2"] != "v2" {
		t.Error("k2 should be untouched")
	}
}

func TestCacheOp_UnknownAction(t *testing.T) {
	deps := newFakeCache()
	_, err := runCache(deps, "flush", "memory", nil, nil)
	if err == nil {
		t.Fatal("expected error for unknown action")
	}
}

func TestCacheOp_DepError(t *testing.T) {
	deps := &fakeCacheDeps{err: fmt.Errorf("redis unavailable")}
	_, err := runCache(deps, "set", "redis",
		map[string]any{"data": map[string]any{"k": "v"}, "ttl": "60"}, nil)
	if err == nil {
		t.Fatal("expected error when dep fails")
	}
}
