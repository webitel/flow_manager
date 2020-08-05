package call

import (
	"context"
	"github.com/webitel/flow_manager/flow"
	"github.com/webitel/flow_manager/model"
)

func (r *Router) amd(ctx context.Context, scope *flow.Flow, call model.Call, args interface{}) (model.Response, *model.AppError) {
	var argv model.AmdParameters

	if err := r.Decode(scope, args, &argv); err != nil {
		return nil, err
	}

	return call.Amd(ctx, argv)
}
