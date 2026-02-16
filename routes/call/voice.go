package call

import (
	"context"

	"github.com/webitel/flow_manager/flow"
	"github.com/webitel/flow_manager/model"
)

type BackgroundPlayback struct {
	Name            string              `json:"name,omitempty"`
	File            *model.PlaybackFile `json:"file" json:"file,omitempty"`
	VolumeReduction int                 `json:"volumeReduction" json:"volume_reduction,omitempty"`
}

type BackgroundPlaybackStop struct {
	Name string `json:"name,omitempty"`
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
		return call.BackgroundPlayback(ctx, res[0], argv.Name, argv.VolumeReduction)
	}
}

func (r *Router) backgroundPlaybackStop(ctx context.Context, scope *flow.Flow, call model.Call, args interface{}) (model.Response, *model.AppError) {
	var argv BackgroundPlaybackStop

	if err := r.Decode(scope, args, &argv); err != nil {
		return nil, err
	}

	return call.BackgroundPlaybackStop(ctx, argv.Name)
}
