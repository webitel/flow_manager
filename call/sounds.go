package call

import (
	"context"
	"github.com/webitel/flow_manager/flow"
	"github.com/webitel/flow_manager/model"
)

type SoundsArgs struct {
	Voice string
	Lang  string
}

func (r *Router) SetSounds(ctx context.Context, scope *flow.Flow, call model.Call, args interface{}) (model.Response, *model.AppError) {
	var argv SoundsArgs
	if err := r.Decode(call, args, &argv); err != nil {
		return nil, err
	}

	if argv.Lang == "" {
		return nil, ErrorRequiredParameter("setSounds", "lang")
	}

	if argv.Voice == "" {
		return nil, ErrorRequiredParameter("setSounds", "voice")
	}

	return call.SetSounds(ctx, argv.Lang, argv.Voice)
}
