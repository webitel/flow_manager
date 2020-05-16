package call

import (
	"context"
	"github.com/webitel/flow_manager/flow"
	"github.com/webitel/flow_manager/model"
)

type RingBackArgs struct {
	All      bool
	Call     *model.PlaybackFile
	Hold     *model.PlaybackFile
	Transfer *model.PlaybackFile
}

func (r *Router) RingBack(ctx context.Context, scope *flow.Flow, call model.Call, args interface{}) (model.Response, *model.AppError) {
	var argv RingBackArgs

	if err := r.Decode(scope, args, &argv); err != nil {
		return nil, err
	}

	// TODO FIXME !!!
	return model.CallResponseError, nil
}
