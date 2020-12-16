package chat_route

import (
	"context"
	"github.com/webitel/flow_manager/flow"
	"github.com/webitel/flow_manager/model"
)

type ExportArg []string

func (r *Router) export(ctx context.Context, scope *flow.Flow, conv Conversation, args interface{}) (model.Response, *model.AppError) {
	var argv ExportArg

	if err := r.Decode(scope, args, &argv); err != nil {
		return nil, err
	}
	return conv.Export(ctx, argv)
}
