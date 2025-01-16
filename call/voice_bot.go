package call

import (
	"context"
	"github.com/webitel/flow_manager/flow"
	"github.com/webitel/flow_manager/model"
)

type VoiceBot struct {
	Connection       string            `json:"connection"`
	Rate             string            `json:"rate"`
	InitialAiMessage string            `json:"initialAiMessage"`
	Variables        map[string]string `json:"variables"`
}

func (r *Router) voiceBot(ctx context.Context, scope *flow.Flow, call model.Call, args interface{}) (model.Response, *model.AppError) {
	var argv VoiceBot
	if err := r.Decode(scope, args, &argv); err != nil {
		return nil, err
	}

	rate := 0

	switch argv.Rate {
	case "8kHz":
		rate = 8000
	case "16kHz":
		rate = 16000
	}

	if argv.Connection == "" {
		return model.CallResponseError, ErrorRequiredParameter("voiceBot", "connection")
	}
	return call.Bot(ctx, argv.Connection, rate, argv.InitialAiMessage, argv.Variables)
}
