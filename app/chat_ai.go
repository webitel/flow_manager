package app

import (
	"context"
	"github.com/webitel/flow_manager/chat_ai"
	"github.com/webitel/flow_manager/model"
	"strings"
	"time"
)

const SysConnectionName = "chat_ai_connection"

var aiConnections = chat_ai.NewHub()

func (fm *FlowManager) ChatAnswerAi(ctx context.Context, domainId int64, r model.ChatAiAnswer) (*chat_ai.MessageResponse, error) {
	if r.Connection == "" {
		var appErr *model.AppError
		r.Connection, appErr = fm.GetSystemSettingsString(ctx, domainId, SysConnectionName)
		if appErr != nil {
			return nil, appErr
		}
	}

	if r.Connection == "" {
		return nil, model.NewRequestError("app.chat_ai.answer", "connection is required")
	}

	cli, err := aiConnections.GetClient(r.Connection)
	if err != nil {
		return nil, err
	}

	cat := strings.Split(r.Categories, ",")

	request := &chat_ai.MessageRequest{
		UserMetadata: r.Variables,
		Categories:   cat,
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
