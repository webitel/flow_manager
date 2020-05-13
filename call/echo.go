package call

import (
	"context"
	"github.com/webitel/flow_manager/flow"
	"github.com/webitel/flow_manager/model"
)

type EchoArg int

func (r *Router) echo(ctx context.Context, scope *flow.Flow, call model.Call, args interface{}) (model.Response, *model.AppError) {
	var delay int
	if err := r.Decode(call, args, &delay); err != nil {
		return nil, err
	}

	return call.Echo(ctx, delay)
}
