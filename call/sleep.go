package call

import "github.com/webitel/flow_manager/model"

func (r *Router) sleep(call model.Call, args interface{}) (model.Response, *model.AppError) {
	var timeout = 0
	return call.Sleep(timeout)
}
