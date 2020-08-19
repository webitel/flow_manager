package grpc

import (
	"context"
	"errors"
	"github.com/webitel/flow_manager/providers/grpc/flow"
)

type chatApi struct {
	*server
}

func NewChatApi(s *server) *chatApi {
	return &chatApi{s}
}

func (s *chatApi) Start(context.Context, *flow.StartRequest) (*flow.StartResponse, error) {
	return nil, errors.New("TODO")
}

func (s *chatApi) Break(context.Context, *flow.BreakRequest) (*flow.BreakResponse, error) {
	return nil, errors.New("TODO")
}

func (s *chatApi) ConfirmationMessage(context.Context, *flow.ConfirmationMessageRequest) (*flow.ConfirmationMessageResponse, error) {
	return nil, errors.New("TODO")
}
