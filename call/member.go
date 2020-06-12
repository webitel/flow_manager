package call

import (
	"context"
	"github.com/webitel/flow_manager/flow"
	"github.com/webitel/flow_manager/model"
)

type queuePosition struct {
	Set string `json:"set"`
}

func (r *Router) QueueCallPosition(ctx context.Context, scope *flow.Flow, call model.Call, args interface{}) (model.Response, *model.AppError) {
	var argv queuePosition

	if err := r.Decode(scope, args, &argv); err != nil {
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
