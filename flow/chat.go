package flow

import (
	"context"

	"github.com/webitel/flow_manager/model"
)

func (r *router) broadcastChatMessage(ctx context.Context, scope *Flow, conn model.Connection, args interface{}) (model.Response, *model.AppError) {
	var err *model.AppError
	var argv = model.BroadcastChat{
		Type: "text",
	}

	if err = scope.Decode(args, &argv); err != nil {
		return nil, err
	}

	// todo add search file
	argv.File = nil

	responseCode := ""

	if err = r.fm.BroadcastChatMessage(ctx, conn.DomainId(), argv); err != nil {
		responseCode = err.Error()
	}

	return conn.Set(ctx, model.Variables{
		argv.ResponseCode: responseCode,
	})
}
