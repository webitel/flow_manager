package chat_route

import (
	"context"

	"github.com/webitel/flow_manager/flow"
	"github.com/webitel/flow_manager/model"
)

type TextMessage string

type ChatAction struct {
	Action model.ChatAction
}

func (r *Router) sendText(ctx context.Context, scope *flow.Flow, conv Conversation, args interface{}) (model.Response, *model.AppError) {
	var argv TextMessage

	if err := r.Decode(scope, args, &argv); err != nil {
		return nil, err
	}

	return conv.SendTextMessage(ctx, string(argv))
}

func (r *Router) sendAction(ctx context.Context, scope *flow.Flow, conv Conversation, args interface{}) (model.Response, *model.AppError) {
	var argv ChatAction
	var err *model.AppError

	if err = r.Decode(scope, args, &argv); err != nil {
		return nil, err
	}

	if err = r.fm.SenChatAction(ctx, conv.Id(), argv.Action); err != nil {
		return model.CallResponseError, err
	}

	return model.CallResponseOK, nil

}
