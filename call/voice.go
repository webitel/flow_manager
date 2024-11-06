package call

import (
	"context"

	"github.com/webitel/flow_manager/flow"
	"github.com/webitel/flow_manager/model"
)

func (r *Router) voice(ctx context.Context, scope *flow.Flow, call model.Call, args interface{}) (model.Response, *model.AppError) {
	// todo
	return call.Hangup(ctx, "")
}

func (r *Router) backgroundPlayback(ctx context.Context, scope *flow.Flow, call model.Call, args interface{}) (model.Response, *model.AppError) {
	//  /opt/webitel/noise-drum-loop.wav
	//return call.BackgroundPlayback("http_cache://http://10.9.8.111:10021/sys/media/539/stream?domain_id=1&.wav")
	return call.BackgroundPlayback("/opt/webitel/noise-drum-loop.wav")
}
