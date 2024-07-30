package app

import (
	"context"
	"github.com/webitel/flow_manager/chat_ai"
	"github.com/webitel/flow_manager/model"
	"time"
)

var aiConnections = chat_ai.NewHub()

func (fm *FlowManager) ChatAnswerAi(ctx context.Context, domainId int64, r model.ChatAiAnswer) (*chat_ai.MessageResponse, error) {
	cli, err := aiConnections.GetClient(r.Connection)
	if err != nil {
		return nil, err
	}

	request := &chat_ai.MessageRequest{
		UserMetadata: r.Variables,
		Categories:   r.Categories,
		Messages:     make([]*chat_ai.Message, 0, len(r.Messages)),
		ModelName:    r.Model,
	}

	for _, v := range r.Messages {
		request.Messages = append(request.Messages, &chat_ai.Message{
			Message: v.Text,
			Sender:  aiChatSender(v.User),
		})
	}
	rctx := ctx
	if r.Timeout > 0 {
		var cancel context.CancelFunc
		rctx, cancel = context.WithTimeout(ctx, time.Duration(r.Timeout)*time.Second)
		defer cancel()
	}

	result, err := cli.Api().Answer(rctx, request)
	if err != nil {
		return nil, err
	}

	return result, nil
}

func aiChatSender(userName string) string {
	if userName != "" {
		return "human"
	}

	return "ai"
}
