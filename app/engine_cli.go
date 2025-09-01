package app

import (
	"context"
	"github.com/webitel/flow_manager/gen/engine"
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

func (fm *FlowManager) GenerateFeedback(ctx context.Context, domainId int64, sourceId string, source string, payload map[string]string) (string, *model.AppError) {
	if fm.engineFeedbackCli == nil {
		return "", model.NewAppError("App", "GenerateFeedback", nil, "engine client not initialized to generate feedback", http.StatusInternalServerError)
	}

	res, err := fm.engineFeedbackCli.Api.GenerateFeedback(ctx, &engine.GenerateFeedbackRequest{
		DomainId: domainId,
		SourceId: sourceId,
		Source:   source,
		Payload:  payload,
	})
	if err != nil {
		return "", model.NewAppError("App", "GenerateFeedback", nil, err.Error(), http.StatusInternalServerError)
	}

	return res.Key, nil
}
