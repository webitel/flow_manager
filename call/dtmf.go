package call

import "github.com/webitel/flow_manager/model"

func (r *Router) dtmfFlush(call model.Call, args interface{}) (model.Response, *model.AppError) {
	return call.FlushDTMF()
}

func (r *Router) inBandDTMF(call model.Call, args interface{}) (model.Response, *model.AppError) {
	req, ok := args.(string)
	if ok && req == "stop" {
		return call.StopDTMF()
	}
	return call.StartDTMF()
}
