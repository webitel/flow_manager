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

	search := make([]*model.PlaybackFile, 0, 3)
	search = append(search, argv.Call, argv.Hold, argv.Transfer)
	if res, err := r.fm.GetMediaFiles(call.DomainId(), &search); err != nil {
		return nil, err
	} else {
		return call.Ringback(ctx, argv.All, res[0], res[1], res[2])
	}
}
