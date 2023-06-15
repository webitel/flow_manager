package flow

import (
	"context"

	"github.com/webitel/flow_manager/model"
)

type ChatHistoryArgs struct {
	ConversationId string `json:"conversationId,omitempty"`
	Variable       string `json:"variable,omitempty"`
	Format         string `json:"format,omitempty"`
	Timeout        string `json:"timeout,omitempty"`
	Limit          string
}

func (r *router) chatHistory(ctx context.Context, scope *Flow, c model.Connection, args interface{}) (model.Response, *model.AppError) {
	var params ChatHistoryArgs
	if err := scope.Decode(args, &params); err != nil {
		return nil, err
	}

	return ResponseOK, nil
}
