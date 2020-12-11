package chat_route

import (
	"context"
	"github.com/webitel/flow_manager/flow"
	"github.com/webitel/flow_manager/model"
)

type FileMessage struct {
	Url string
}

func (r *Router) sendFile(ctx context.Context, scope *flow.Flow, conv Conversation, args interface{}) (model.Response, *model.AppError) {
	var argv FileMessage

	if err := r.Decode(scope, args, &argv); err != nil {
		return nil, err
	}

	return conv.SendImageMessage(ctx, argv.Url)
}
