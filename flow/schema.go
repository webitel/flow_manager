package flow

import (
	"context"
	"github.com/webitel/flow_manager/model"
)

type SchemaArgs struct {
	Id    int  `json:"id"`
	Async bool `json:"async"`
}

func (r *router) schema(ctx context.Context, scope *Flow, c model.Connection, args interface{}) (model.Response, *model.AppError) {
	var argv SchemaArgs
	var schema *model.Schema
	var err *model.AppError

	if err = scope.Decode(args, &argv); err != nil {
		return nil, err
	}

	if schema, err = r.fm.GetSchemaById(c.DomainId(), argv.Id); err != nil {
		return nil, err
	}

	parent := scope.Fork(schema.Name, schema.Schema)

	if argv.Async {
		go Route(ctx, parent, scope.handler)
	} else {
		Route(ctx, parent, scope.handler)
		if parent.IsCancel() {
			scope.SetCancel()
		}
	}

	return ResponseOK, nil
}
