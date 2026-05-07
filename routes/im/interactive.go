package im

import (
	"context"

	"github.com/webitel/flow_manager/flow"
	"github.com/webitel/flow_manager/model"
)

func (router *Router) SendInteractive(ctx context.Context, scope *flow.Flow, conv Dialog, args any) (model.Response, *model.AppError) {
	var interactivePayload model.SendInteractiveRequest
	if err := router.Decode(scope, args, &interactivePayload); err != nil {
		return nil, err
	}

	response, err := conv.SendInteractive(ctx, interactivePayload)
	if err != nil {
		return nil, err
	}

	return response, nil
}
