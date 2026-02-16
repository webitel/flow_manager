package call

import (
	"context"
	"github.com/webitel/flow_manager/flow"
	"github.com/webitel/flow_manager/model"
)

func (r *Router) ringReady(ctx context.Context, scope *flow.Flow, call model.Call, args interface{}) (model.Response, *model.AppError) {
	return call.RingReady(ctx)
}

func (r *Router) preAnswer(ctx context.Context, scope *flow.Flow, call model.Call, args interface{}) (model.Response, *model.AppError) {
	return call.PreAnswer(ctx)
}

func (r *Router) answer(ctx context.Context, scope *flow.Flow, call model.Call, args interface{}) (model.Response, *model.AppError) {
	return call.Answer(ctx)
}
