package call

import "github.com/webitel/flow_manager/model"

func (r *Router) hangup(call model.Call, args interface{}) (model.Response, *model.AppError) {

	return call.Hangup("")
}
