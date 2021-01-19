package flow

import (
	"context"
	"github.com/webitel/flow_manager/model"
)

type queuePosition struct {
	Set string `json:"set"`
}

func (r *router) QueueCallPosition(ctx context.Context, scope *Flow, call model.Connection, args interface{}) (model.Response, *model.AppError) {
	var argv queuePosition

	if err := scope.Decode(args, &argv); err != nil {
		return nil, err
	}

	if argv.Set == "" {
		return nil, ErrorRequiredParameter("queueCallPosition", "SET")
	}

	pos, err := r.fm.GetCallPosition(call.Id())
	if err != nil {
		return nil, err
	}

	return call.Set(ctx, model.Variables{
		argv.Set: pos,
	})
}
