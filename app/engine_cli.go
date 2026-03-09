package app

import (
	"context"
	"net/http"

	"github.com/webitel/flow_manager/gen/engine"
	"github.com/webitel/flow_manager/model"
)

func (fm *FlowManager) MakeCall(ctx context.Context, req model.OutboundCallRequest) (string, *model.AppError) {
	if fm.engineCallCli == nil {
		return "", model.NewAppError("App", "MakeCall", nil, "engine client not initialized to make a call", http.StatusInternalServerError)
	}

	protoReq, err := parseOutboundCallRequest(&req)
	if err != nil {
		return "", model.NewAppError("App", "MakeCall", nil, err.Error(), http.StatusBadRequest)
	}

	res, err := fm.engineCallCli.Api.CreateCallNA(ctx, protoReq)
	if err != nil {
		return "", model.NewAppError("App", "MakeCall", nil, err.Error(), http.StatusInternalServerError)
	}

	return res.Id, nil
}

func parseOutboundCallRequest(req *model.OutboundCallRequest) (*engine.CreateCallRequest, error) {
	protoReq := &engine.CreateCallRequest{
		Destination: req.Destination,
		DomainId:    req.DomainID,
	}
	if req.From != nil {
		protoReq.From = &engine.CreateCallRequest_EndpointRequest{
			AppId:     req.From.AppId,
			Type:      req.From.Type,
			Id:        req.From.Id,
			Extension: req.From.Extension,
		}
	}

	if req.To != nil {
		protoReq.From = &engine.CreateCallRequest_EndpointRequest{
			AppId:     req.To.AppId,
			Type:      req.To.Type,
			Id:        req.To.Id,
			Extension: req.To.Extension,
		}
	}

	if req.Params != nil {
		protoReq.Params = &engine.CreateCallRequest_CallSettings{
			Timeout:           req.Params.Timeout,
			Variables:         req.Params.Variables,
			Display:           req.Params.Display,
			DisableStun:       req.Params.DisableStun,
			CancelDistribute:  req.Params.CancelDistribute,
			IsOnline:          req.Params.IsOnline,
			DisableAutoAnswer: req.Params.DisableAutoAnswer,
			HideNumber:        req.Params.HideNumber,
		}
	}
	return protoReq, nil
}

func (fm *FlowManager) GenerateFeedback(ctx context.Context, domainId int64, sourceId, source string, payload map[string]string) (string, *model.AppError) {
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
