package call

import (
	"context"

	"github.com/webitel/flow_manager/flow"
	"github.com/webitel/flow_manager/model"
)

type BackgroundPlayback struct {
	File            *model.PlaybackFile `json:"file"`
	VolumeReduction int                 `json:"volumeReduction"`
}

func (r *Router) backgroundPlayback(ctx context.Context, scope *flow.Flow, call model.Call, args interface{}) (model.Response, *model.AppError) {
	var argv BackgroundPlayback

	if err := r.Decode(scope, args, &argv); err != nil {
		return nil, err
	}

	search := make([]*model.PlaybackFile, 0, 1)
	search = append(search, argv.File)
	if res, err := r.fm.GetMediaFiles(call.DomainId(), &search); err != nil {
		return nil, err
	} else {
		return call.BackgroundPlayback(ctx, res[0], argv.VolumeReduction)
	}
}
