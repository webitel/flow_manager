package app

import (
	"context"
	"github.com/webitel/flow_manager/model"
	"net/http"
)

func (fm *FlowManager) MakeCall(ctx context.Context, req model.OutboundCallRequest) (string, *model.AppError) {
	if fm.engineCallCli == nil {
		return "", model.NewAppError("App", "MakeCall", nil, "engine client not initialized to make a call", http.StatusInternalServerError)
	}
	res, err := fm.engineCallCli.Api.CreateCallNA(ctx, req)
	if err != nil {
		return "", model.NewAppError("App", "MakeCall", nil, err.Error(), http.StatusInternalServerError)
	}

	return res.Id, nil
}
