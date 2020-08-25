package chat_route

import (
	"context"
	"github.com/webitel/flow_manager/flow"
	"github.com/webitel/flow_manager/model"
)

type TextMessage string

func (r *Router) sendText(ctx context.Context, scope *flow.Flow, conv Conversation, args interface{}) (model.Response, *model.AppError) {
	var argv TextMessage

	if err := r.Decode(scope, args, &argv); err != nil {
		return nil, err
	}

	return conv.SendTextMessage(ctx, string(argv))
}
