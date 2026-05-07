package flow

import (
	"context"
	"net/http"
	"time"

	"github.com/webitel/flow_manager/model"
)

type ChatHistoryArgs struct {
	ConversationId string `json:"conversationId,omitempty"`
	Variable       string `json:"variable,omitempty"`
	Format         string `json:"format,omitempty"`
	Timeout        int    `json:"timeout,omitempty"`
	Limit          int    `json:"limit,omitempty"`
}

func (r *router) chatHistory(ctx context.Context, scope *Flow, conn model.Connection, args interface{}) (model.Response, *model.AppError) {
	var params = ChatHistoryArgs{
		ConversationId: conn.Id(),
	}
	if appErr := scope.Decode(args, &params); appErr != nil {
		return nil, appErr
	}
	if params.Limit == 0 {
		params.Limit = 300
	}
	if params.Timeout == 0 {
		params.Timeout = 3000
	}
	ctx, _ = context.WithTimeout(ctx, time.Millisecond*time.Duration(params.Timeout))
	messages, messagesErr := r.fm.GetChatMessagesByConversationId(ctx, conn.DomainId(), params.ConversationId, int64(params.Limit))
	if messagesErr != nil {
		return nil, model.NewAppError("chatHistory", "chat.get_messages", nil, messagesErr.Error(), http.StatusInternalServerError)
	}
	text, parseErr := r.fm.ParseChatMessages(messages, params.Format)
	if parseErr != nil {
		return nil, model.NewAppError("chatHistory", "chat.parse_messages", nil, parseErr.Error(), http.StatusInternalServerError)
	}
	return conn.Set(ctx, model.Variables{params.Variable: text})
}
