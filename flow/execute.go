package flow

import (
	"context"
	"github.com/webitel/flow_manager/model"
)

type executeArgs struct {
	flow *Flow
}

func (r *router) execute(ctx context.Context, scope *Flow, conn model.Connection, args interface{}) (model.Response, *model.AppError) {
	return ResponseOK, nil
}
