package call

import (
	"context"

	"github.com/webitel/flow_manager/flow"
	"github.com/webitel/flow_manager/model"
)

type PickupArgs struct {
	Name string
}

func (r *Router) pickup(ctx context.Context, scope *flow.Flow, call model.Call, args interface{}) (model.Response, *model.AppError) {
	var argv PickupArgs

	if err := r.Decode(scope, args, &argv); err != nil {
		return nil, err
	}

	if argv.Name == "" {
		return model.CallResponseError, ErrorRequiredParameter("pickup", "name")
	}

	return call.Pickup(ctx, argv.Name)
}
