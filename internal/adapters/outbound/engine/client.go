package engine

import (
	"context"
	"fmt"
	"sync"

	"github.com/webitel/wlog"

	engine2 "github.com/webitel/flow_manager/api/gen/engine"
	"github.com/webitel/flow_manager/internal/domain/call"
	"github.com/webitel/flow_manager/internal/infrastructure/grpcdial"
)

const serviceName = "engine"

type Client struct {
	consulAddr string
	startOnce  sync.Once
	call       *grpcdial.Client[engine2.CallServiceClient]
	feedback   *grpcdial.Client[engine2.FeedbackServiceClient]
}

func New(consulAddr string) *Client {
	return &Client{consulAddr: consulAddr}
}

func (c *Client) Start() error {
	wlog.Debug("starting engine client")
	var err error
	c.startOnce.Do(func() {
		c.call, err = grpcdial.NewClient(c.consulAddr, serviceName, engine2.NewCallServiceClient)
		if err != nil {
			return
		}
		c.feedback, err = grpcdial.NewClient(c.consulAddr, serviceName, engine2.NewFeedbackServiceClient)
	})
	return err
}

func (c *Client) Stop() {}

func (c *Client) MakeCall(ctx context.Context, req call.OutboundCallRequest) (string, error) {
	protoReq := &engine2.CreateCallRequest{
		Destination: req.Destination,
		DomainId:    req.DomainID,
	}
	if req.From != nil {
		protoReq.From = &engine2.CreateCallRequest_EndpointRequest{
			AppId:     req.From.AppId,
			Type:      req.From.Type,
			Id:        req.From.Id,
			Extension: req.From.Extension,
		}
	}
	if req.To != nil {
		protoReq.To = &engine2.CreateCallRequest_EndpointRequest{
			AppId:     req.To.AppId,
			Type:      req.To.Type,
			Id:        req.To.Id,
			Extension: req.To.Extension,
		}
	}
	if req.Params != nil {
		protoReq.Params = &engine2.CreateCallRequest_CallSettings{
			Timeout:           req.Params.Timeout,
			Variables:         req.Params.Variables,
			Display:           req.Params.Display,
			DisableStun:       req.Params.DisableStun,
			CancelDistribute:  req.Params.CancelDistribute,
			IsOnline:          req.Params.IsOnline,
			DisableAutoAnswer: req.Params.DisableAutoAnswer,
			HideNumber:        req.Params.HideNumber,
			ContactId:         req.Params.ContactId,
		}
	}

	res, err := c.call.API.CreateCallNA(ctx, protoReq)
	if err != nil {
		return "", fmt.Errorf("engine.MakeCall: %w", err)
	}
	return res.Id, nil
}

func (c *Client) GenerateFeedback(ctx context.Context, domainId int64, sourceId, source string, payload map[string]string) (string, error) {
	res, err := c.feedback.API.GenerateFeedback(ctx, &engine2.GenerateFeedbackRequest{
		DomainId: domainId,
		SourceId: sourceId,
		Source:   source,
		Payload:  payload,
	})
	if err != nil {
		return "", fmt.Errorf("engine.GenerateFeedback: %w", err)
	}
	return res.Key, nil
}
