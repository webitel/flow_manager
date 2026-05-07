package builtin

import (
	"context"
	"fmt"
	"strconv"

	"github.com/webitel/flow_manager/internal/runtime/ops"
)

// CacheDeps is the narrow interface required by the cache op.
type CacheDeps interface {
	CacheGet(ctx context.Context, cacheType string, domainID int64, key string) (string, error)
	CacheSet(ctx context.Context, cacheType string, domainID int64, key string, value string, ttlSecs int64) error
	CacheDelete(ctx context.Context, cacheType string, domainID int64, key string) error
}

type cacheOp struct{ deps CacheDeps }

// CacheOp returns the native cache op: get/set/delete keyed values in memory or
// redis cache, scoped to domainID.
func CacheOp(deps CacheDeps) ops.Op { return cacheOp{deps: deps} }

func (cacheOp) Kind() ops.OpKind { return ops.OpKindSync }

func (o cacheOp) Execute(ctx context.Context, in ops.OpInput) (ops.OpOutput, error) {
	cacheType, _ := in.Node.Args["type"].(string)
	action, _ := in.Node.Args["action"].(string)

	switch action {
	case "set":
		setObj, ok := in.Node.Args["set"].(map[string]any)
		if !ok {
			return ops.OpOutput{}, fmt.Errorf("cache: 'set' object required for action=set")
		}
		ttlStr, _ := setObj["ttl"].(string)
		ttl, err := strconv.ParseInt(ttlStr, 10, 64)
		if err != nil {
			return ops.OpOutput{}, fmt.Errorf("cache: invalid ttl %q: %w", ttlStr, err)
		}
		dataRaw, _ := setObj["data"].(map[string]any)
		for key, val := range dataRaw {
			value := ops.ExpandStr(fmt.Sprintf("%v", val), in.Variables, in.GlobalVar)
			if err := o.deps.CacheSet(ctx, cacheType, in.DomainID, key, value, ttl); err != nil {
				return ops.OpOutput{}, fmt.Errorf("cache set %q: %w", key, err)
			}
		}

	case "get":
		getMap, ok := in.Node.Args["get"].(map[string]any)
		if !ok {
			return ops.OpOutput{}, fmt.Errorf("cache: 'get' object required for action=get")
		}
		vars := make(map[string]string, len(getMap))
		for varName, rawKey := range getMap {
			cacheKey := ops.ExpandStr(fmt.Sprintf("%v", rawKey), in.Variables, in.GlobalVar)
			val, err := o.deps.CacheGet(ctx, cacheType, in.DomainID, cacheKey)
			if err != nil {
				return ops.OpOutput{}, fmt.Errorf("cache get %q: %w", cacheKey, err)
			}
			vars[varName] = val
		}
		return ops.OpOutput{SetVars: vars}, nil

	case "delete":
		deleteObj, ok := in.Node.Args["delete"].(map[string]any)
		if !ok {
			return ops.OpOutput{}, fmt.Errorf("cache: 'delete' object required for action=delete")
		}
		keysRaw, _ := deleteObj["keys"].([]any)
		for _, k := range keysRaw {
			key := ops.ExpandStr(fmt.Sprintf("%v", k), in.Variables, in.GlobalVar)
			if err := o.deps.CacheDelete(ctx, cacheType, in.DomainID, key); err != nil {
				return ops.OpOutput{}, fmt.Errorf("cache delete %q: %w", key, err)
			}
		}

	default:
		return ops.OpOutput{}, fmt.Errorf("cache: unknown action %q", action)
	}

	return ops.OpOutput{}, nil
}
