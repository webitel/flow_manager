package call

import (
	"context"
	"github.com/webitel/flow_manager/flow"
	"github.com/webitel/flow_manager/model"
)

type SleepArgs int

func (r *Router) sleep(ctx context.Context, scope *flow.Flow, call model.Call, args interface{}) (model.Response, *model.AppError) {
	var timeout int
	if err := r.Decode(call, args, &timeout); err != nil {
		return nil, err
	}
	return call.Sleep(ctx, timeout)
}
