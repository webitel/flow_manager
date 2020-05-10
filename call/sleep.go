package call

import (
	"github.com/webitel/flow_manager/model"
)

type SleepArgs int

func (r *Router) sleep(call model.Call, args interface{}) (model.Response, *model.AppError) {
	var timeout int
	if err := r.Decode(call, args, &timeout); err != nil {
		return nil, err
	}
	return call.Sleep(timeout)
}
