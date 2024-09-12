package flow

import (
	"context"
	"github.com/webitel/flow_manager/model"
)

type LogArg string

func (r *router) Log(ctx context.Context, scope *Flow, conn model.Connection, args interface{}) (model.Response, *model.AppError) {
	var log LogArg
	if err := scope.Decode(args, &log); err != nil {
		return nil, err
	} else {
		scope.log.Info(string(log))
		return model.CallResponseOK, nil
	}
}
