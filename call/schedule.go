package call

import (
	"context"
	"github.com/webitel/flow_manager/flow"
	"github.com/webitel/flow_manager/model"
)

type ScheduleHangupArgs struct {
	Seconds int
	Cause   string
}

func (r *Router) ScheduleHangup(ctx context.Context, scope *flow.Flow, call model.Call, args interface{}) (model.Response, *model.AppError) {
	var argv = ScheduleHangupArgs{
		Seconds: 2,
		Cause:   "",
	}

	if err := r.Decode(scope, args, &argv); err != nil {
		return nil, err
	}

	return call.ScheduleHangup(ctx, argv.Seconds, argv.Cause)
}
