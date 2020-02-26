package app

import (
	"fmt"
	"github.com/webitel/flow_manager/model"
	"github.com/webitel/wlog"
)

func (f *FlowManager) GetSchema(domainId, id int, updatedAt int64) (schema *model.Schema, err *model.AppError) {
	if v, ok := f.schemaCache.Get(id); ok {
		schema = v.(*model.Schema)
		if schema.UpdatedAt == updatedAt {
			return schema, nil
		}
	}

	if schema, err = f.Store.Schema().Get(domainId, id); err != nil {
		return
	}

	wlog.Debug(fmt.Sprintf("add schema \"%s\" [%d] to cache", schema.Name, schema.Id))
	f.schemaCache.AddWithDefaultExpires(id, schema)
	return
}
