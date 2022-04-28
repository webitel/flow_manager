package processing

import (
	"context"
	"github.com/webitel/flow_manager/flow"
	"github.com/webitel/flow_manager/model"
)

type FromArgs struct {
	Set     string                 `json:"set"`
	Actions []string               `json:"actions"`
	View    map[string]interface{} `json:"view"`
}

func (r *Router) form(ctx context.Context, scope *flow.Flow, conn Connection, args interface{}) (model.Response, *model.AppError) {
	var argv FromArgs
	var err *model.AppError

	if err = r.Decode(scope, args, &argv); err != nil {
		return nil, err
	}

	var action *model.FormAction
	action, err = conn.PushForm(argv.Actions, argv.View)
	if err != nil {
		return nil, err
	}

	return conn.Set(ctx, action.Fields)
}
