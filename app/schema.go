package app

import (
	"fmt"
	"net/http"

	"golang.org/x/sync/singleflight"

	"github.com/webitel/flow_manager/model"
	"github.com/webitel/wlog"
)

var requestGroup singleflight.Group

func (f *FlowManager) GetSchema(domainId int64, id int, updatedAt int64) (schema *model.Schema, err *model.AppError) {
	var v interface{}
	var ok bool
	var doErr error

	if v, ok = f.schemaCache.Get(id); ok {
		schema = v.(*model.Schema)
		if schema.UpdatedAt == updatedAt {
			return schema, nil
		}
	}

	v, doErr, _ = requestGroup.Do(fmt.Sprintf("GetSchema-%d-%d-%d", domainId, id, updatedAt), func() (interface{}, error) {
		res, appErr := f.Store.Schema().Get(domainId, id)
		if appErr != nil {
			return nil, appErr
		}

		return res, nil
	})

	if doErr != nil {
		switch doErr.(type) {
		case *model.AppError:
			err = doErr.(*model.AppError)
		default:
			err = model.NewAppError("App", "app.get_schema.app_err", nil, doErr.Error(), http.StatusInternalServerError)
		}

		return
	}

	schema = v.(*model.Schema)

	wlog.Debug(fmt.Sprintf("add schema \"%s\" [%d] to cache", schema.Name, schema.Id))
	f.schemaCache.AddWithDefaultExpires(id, schema)
	return
}

func (f *FlowManager) GetSchemaById(domainId int64, id int) (*model.Schema, *model.AppError) {

	res, err, _ := requestGroup.Do(fmt.Sprintf("GetSchemaById-%d-%d", domainId, id), func() (interface{}, error) {
		updatedAt, err := f.Store.Schema().GetUpdatedAt(domainId, id)
		if err != nil {
			return nil, err
		}

		return updatedAt, nil
	})

	if err != nil {
		switch err.(type) {
		case *model.AppError:
			return nil, err.(*model.AppError)
		default:
			return nil, model.NewAppError("App", "app.get_schema_by_id.app_err", nil, err.Error(), http.StatusInternalServerError)
		}
	}

	return f.GetSchema(domainId, id, res.(int64))
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
