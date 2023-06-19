package flow

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/webitel/flow_manager/model"
)

type ChatHistoryArgs struct {
	ConversationId string `json:"conversationId,omitempty"`
	Variable       string `json:"variable,omitempty"`
	Format         string `json:"format,omitempty"`
	Timeout        string `json:"timeout,omitempty"`
	Limit          string `json:"limit,omitempty"`
}

func (r *router) chatHistory(ctx context.Context, scope *Flow, conn model.Connection, args interface{}) (model.Response, *model.AppError) {
	var params ChatHistoryArgs
	if appErr := scope.Decode(args, &params); appErr != nil {
		return nil, appErr
	}

	limitParsed, err := strconv.ParseInt(params.Limit, 10, 0)
	if err != nil {
		return nil, model.NewAppError("ChatHistory", "flow.chat_history.valid_args", nil, "bad arguments", http.StatusBadRequest)
	}
	timeoutParsed, err := strconv.Atoi(params.Timeout)
	if timeoutParsed == 0 || err != nil {
		timeoutParsed = 3000
	}
	ctx, _ = context.WithTimeout(ctx, time.Millisecond*time.Duration(timeoutParsed))
	messages, appErr := r.fm.GetChatMessagesByConversationId(ctx, conn.DomainId(), params.ConversationId, limitParsed)
	if err != nil {
		return nil, appErr
	}
	text, appErr := r.fm.ParseChatMessages(messages, params.Format)
	if appErr != nil {
		return nil, appErr
	}
	return conn.Set(ctx, model.Variables{params.Variable: text})
}
