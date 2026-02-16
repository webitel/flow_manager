package chat

import (
	"context"

	"github.com/webitel/flow_manager/flow"
	"github.com/webitel/flow_manager/model"
)

func (r *Router) Menu(ctx context.Context, scope *flow.Flow, conv Conversation, args any) (model.Response, *model.AppError) {
	var argv model.ChatMenuArgs
	argv.Type = "buttons"

	if err := r.Decode(scope, args, &argv); err != nil {
		return nil, err
	}

	switch argv.Type {
	case "buttons", "inline":

	default:
		return model.CallResponseError, nil
	}

	return conv.SendMenu(ctx, &argv)
}
