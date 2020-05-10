package call

import "github.com/webitel/flow_manager/model"

type BandDTMFArg string

func (r *Router) dtmfFlush(call model.Call, args interface{}) (model.Response, *model.AppError) {
	return call.FlushDTMF()
}

func (r *Router) inBandDTMF(call model.Call, args interface{}) (model.Response, *model.AppError) {
	var conf BandDTMFArg = ""
	if err := r.Decode(call, args, &conf); err != nil {
		return nil, err
	}

	if conf == "stop" {
		return call.StopDTMF()
	}
	return call.StartDTMF()
}
