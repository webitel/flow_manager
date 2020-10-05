package chat_route

import (
	"context"
	"fmt"
	"github.com/webitel/flow_manager/flow"
	"github.com/webitel/flow_manager/model"
	"github.com/webitel/wlog"
)

type Bridge struct {
	UserId int64 `json:"userId"`
}

func (r *Router) bridge(ctx context.Context, scope *flow.Flow, conv Conversation, args interface{}) (model.Response, *model.AppError) {
	var argv Bridge

	if err := r.Decode(scope, args, &argv); err != nil {
		return nil, err
	}

	wlog.Debug(fmt.Sprintf("conversation %d bridge to %d", conv.Id(), argv.UserId))

	if err := conv.Bridge(ctx, argv.UserId); err != nil {
		return nil, err
	}

	return model.CallResponseOK, nil
}
