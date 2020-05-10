package call

import "github.com/webitel/flow_manager/model"

type ScheduleHangupArgs struct {
	Seconds int
	Cause   string
}

func (r *Router) ScheduleHangup(call model.Call, args interface{}) (model.Response, *model.AppError) {
	var argv = ScheduleHangupArgs{
		Seconds: 2,
		Cause:   "",
	}

	if err := r.Decode(call, args, &argv); err != nil {
		return nil, err
	}

	return call.ScheduleHangup(argv.Seconds, argv.Cause)
}
