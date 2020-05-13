package call

import (
	"context"
	"github.com/webitel/flow_manager/flow"
	"github.com/webitel/flow_manager/model"
)

type HangupArg string

func (r *Router) hangup(ctx context.Context, scope *flow.Flow, call model.Call, args interface{}) (model.Response, *model.AppError) {
	var argv HangupArg

	if err := r.Decode(call, args, &argv); err != nil {
		return nil, err
	}

	return call.Hangup(ctx, string(argv))
}
