package flow

import (
	"context"
	"github.com/webitel/flow_manager/model"
)

type GotoArg string

func (r *router) GotoTag(ctx context.Context, scope *Flow, conn model.Connection, args interface{}) (model.Response, *model.AppError) {
	var tag GotoArg
	if err := Decode(conn, args, &tag); err != nil {
		return nil, err
	}
	if !scope.Goto(string(tag)) {
		return ResponseErr, nil
	}
	return ResponseOK, nil
}
