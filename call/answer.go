package call

import (
	"github.com/webitel/flow_manager/model"
)

func (r *Router) ringReady(call model.Call, args interface{}) (model.Response, *model.AppError) {
	return call.RingReady()
}

func (r *Router) preAnswer(call model.Call, args interface{}) (model.Response, *model.AppError) {
	return call.PreAnswer()
}

func (r *Router) answer(call model.Call, args interface{}) (model.Response, *model.AppError) {
	return call.Answer()
}
