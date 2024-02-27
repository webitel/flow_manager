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

func init() {
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

	if sb.Encrypt {
		b, err2 := f.cert.DecryptBytes(sb.Value)
		if err2 != nil {
			wlog.Error(fmt.Sprintf("decrypt schema variable error: %s", err2.Error()))
			return ""
		}
		val := removeQuote(b)
		variableCache.AddWithDefaultExpires(key, val)
		return val
	} else {
		val := removeQuote(sb.Value)
		variableCache.AddWithDefaultExpires(key, val)
		return val
	}
}

func removeQuote(text []byte) string {
	l := len(text)
	if l < 2 {
		return string(text)
	}

	if text[0] == '"' {
		text = text[1:]
		l = l - 1
	}

	if text[l-1] == '"' {
		text = text[:l-1]
	}

	return string(text)
}
