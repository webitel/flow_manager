package flow

import (
	"context"
	"github.com/webitel/flow_manager/model"
)

type GlobalVar struct {
	Value   string `json:"value"`
	Encrypt bool   `json:"encrypt"`
}

type GlobalArgs map[string]GlobalVar

func (r *router) set(ctx context.Context, scope *Flow, conn model.Connection, args interface{}) (model.Response, *model.AppError) {
	var vars model.Variables
	if err := scope.Decode(args, &vars); err != nil {
		return nil, err
	}

	return conn.Set(ctx, vars)
}

func (r *router) global(ctx context.Context, scope *Flow, conn model.Connection, args interface{}) (model.Response, *model.AppError) {
	var argv GlobalArgs
	var err *model.AppError

	if err = scope.Decode(args, &argv); err != nil {
		return nil, err
	}

	vars := make(map[string]*model.SchemaVariable)
	for k, v := range argv {
		vars[k] = &model.SchemaVariable{
			Encrypt: v.Encrypt,
			Value:   []byte(v.Value),
		}
	}

	err = r.fm.SetSchemaVariable(ctx, conn.DomainId(), vars)
	if err != nil {
		return nil, err
	}

	return model.CallResponseOK, nil
}
