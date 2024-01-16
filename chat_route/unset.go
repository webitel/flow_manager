package chat_route

import (
	"context"
	"net/http"

	"github.com/webitel/flow_manager/flow"
	"github.com/webitel/flow_manager/model"
)

type UnSetArg []string

func (r *Router) UnSet(ctx context.Context, scope *flow.Flow, conv Conversation, args interface{}) (model.Response, *model.AppError) {
	var argv UnSetArg

	if err := r.Decode(scope, args, &argv); err != nil {
		return nil, err
	}

	if len(argv) == 0 {
		return nil, model.NewAppError("UnSet", "chat.unset.valid", nil, "bad arguments", http.StatusBadRequest)
	}

	return conv.UnSet(ctx, argv)
}
