package app

import (
	"fmt"
	"github.com/webitel/flow_manager/model"
	"github.com/webitel/wlog"
)

func (f *FlowManager) GetSchema(domainId int64, id int, updatedAt int64) (schema *model.Schema, err *model.AppError) {
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

func (f *FlowManager) GetSchemaById(domainId int64, id int) (*model.Schema, *model.AppError) {
	updatedAt, err := f.Store.Schema().GetUpdatedAt(domainId, id)

	if err != nil {
		return nil, err
	}

	return f.GetSchema(domainId, id, updatedAt)
}

func (f *FlowManager) SearchTransferredRouting(domainId int64, schemaId int) (*model.Routing, *model.AppError) {
	routing, err := f.Store.Schema().GetTransferredRouting(domainId, schemaId)
	if err != nil {
		return nil, err
	}

	routing.Schema, err = f.GetSchema(domainId, routing.SchemaId, routing.SchemaUpdatedAt)
	if err != nil {
		return nil, err
	}

	return routing, nil
}
