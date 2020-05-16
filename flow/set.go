package flow

import (
	"context"
	"github.com/webitel/flow_manager/model"
)

func (r *router) set(ctx context.Context, scope *Flow, conn model.Connection, args interface{}) (model.Response, *model.AppError) {
	var vars model.Variables
	if err := scope.Decode(args, &vars); err != nil {
		return nil, err
	}

	return conn.Set(ctx, vars)
}
