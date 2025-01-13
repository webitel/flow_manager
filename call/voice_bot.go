package call

import (
	"context"
	"github.com/webitel/flow_manager/flow"
	"github.com/webitel/flow_manager/model"
)

type VoiceBot struct {
	Connection string `json:"connection"`
}

func (r *Router) voiceBot(ctx context.Context, scope *flow.Flow, call model.Call, args interface{}) (model.Response, *model.AppError) {
	var argv VoiceBot
	if err := r.Decode(scope, args, &argv); err != nil {
		return nil, err
	}

	if argv.Connection == "" {
		return model.CallResponseError, ErrorRequiredParameter("voiceBot", "connection")
	}
	return call.Bot(ctx, argv.Connection)
}
