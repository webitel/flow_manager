package chat_route

import (
	"context"
	"github.com/webitel/flow_manager/flow"
	"github.com/webitel/flow_manager/model"
	"strings"
)

type ReceiveMessage struct {
	Timeout int
	Set     string
}

func (r *Router) recvMessage(ctx context.Context, scope *flow.Flow, conv Conversation, args interface{}) (model.Response, *model.AppError) {
	var argv ReceiveMessage

	if err := r.Decode(scope, args, &argv); err != nil {
		return nil, err
	}

	if argv.Set == "" {
		return model.CallResponseOK, nil
	}

	msgs, err := conv.ReceiveMessage(ctx, argv.Set, argv.Timeout)
	if err != nil {
		return nil, err
	}

	return conv.Set(ctx, model.Variables{
		argv.Set: strings.Join(msgs, " "),
	})
}
