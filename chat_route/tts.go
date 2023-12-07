package chat_route

import (
	"context"
	"github.com/webitel/flow_manager/flow"
	"github.com/webitel/flow_manager/model"
)

type TTSArgs struct {
	Message   string `json:"message,omitempty"`
	ProfileId int    `json:"profileId,omitempty"`
}

func (r *Router) sendTTS(ctx context.Context, scope *flow.Flow, conv Conversation, args interface{}) (model.Response, *model.AppError) {
	var argv TTSArgs
	var err *model.AppError

	if err = r.Decode(scope, args, &argv); err != nil {
		return nil, err
	}

	uri, err := r.fm.GenerateTTSLink(ctx, argv.Message, conv.DomainId(), argv.ProfileId)
	if err != nil {
		return model.CallResponseError, err
	}

	return conv.SendFile(ctx, "", &model.File{Url: uri})
}
