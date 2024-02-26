package app

import (
	"context"
	"fmt"

	"github.com/webitel/wlog"

	"github.com/webitel/engine/utils"
	"golang.org/x/sync/singleflight"
)

var variableGroup singleflight.Group
var variableCache utils.ObjectCache

func initCache() {
	variableCache = utils.NewLruWithParams(10000, "variable", 10, "")
}

func (f *FlowManager) SchemaVariable(ctx context.Context, domainId int64, name string) string {
	key := fmt.Sprintf("%d-%s", domainId, name)

	v, ok := variableCache.Get(key)
	if ok {
		return v.(string)
	}

	v, _, _ = variableGroup.Do(key, func() (interface{}, error) {
		return f.schemaVariable(key, domainId, name), nil
	})

	return v.(string)
}

func (f *FlowManager) schemaVariable(key string, domainId int64, name string) string {

	sb, err := f.Store.Schema().GetVariable(domainId, name)
	if err != nil {
		wlog.Error(fmt.Sprintf("get schema variable error: %s", err.Error()))
		return ""
	}

	if sb.Encrypted {
		b, err2 := f.cert.DecryptBytes([]byte(sb.Value))
		if err2 != nil {
			wlog.Error(fmt.Sprintf("decrypt schema variable error: %s", err2.Error()))
			return ""
		}
		val := string(b)

		variableCache.AddWithDefaultExpires(key, val)
		return val
	} else {
		variableCache.AddWithDefaultExpires(key, sb.Value)
		return sb.Value
	}
}
