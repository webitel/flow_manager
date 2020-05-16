package call

import (
	"context"
	"github.com/webitel/flow_manager/flow"
	"github.com/webitel/flow_manager/model"
)

type BandDTMFArg string

func (r *Router) dtmfFlush(ctx context.Context, scope *flow.Flow, call model.Call, args interface{}) (model.Response, *model.AppError) {
	return call.FlushDTMF(ctx)
}

func (r *Router) inBandDTMF(ctx context.Context, scope *flow.Flow, call model.Call, args interface{}) (model.Response, *model.AppError) {
	var conf BandDTMFArg = ""
	if err := r.Decode(scope, args, &conf); err != nil {
		return nil, err
	}

	if conf == "stop" {
		return call.StopDTMF(ctx)
	}
	return call.StartDTMF(ctx)
}
