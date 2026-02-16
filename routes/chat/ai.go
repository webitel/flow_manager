package chat

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/webitel/flow_manager/flow"
	"github.com/webitel/flow_manager/model"
)

func (r *Router) chatAi(ctx context.Context, scope *flow.Flow, conv Conversation, args any) (model.Response, *model.AppError) {
	var argv model.ChatAiAnswer

	if err := r.Decode(scope, args, &argv); err != nil {
		return nil, err
	}

	if argv.HistoryLength == 0 {
		argv.HistoryLength = 10
	}

	argv.Messages = conv.LastMessages(argv.HistoryLength)

	res, err := r.fm.ChatAnswerAi(ctx, conv.DomainId(), argv)
	if err != nil {
		return model.CallResponseError, model.NewAppError("chatAi", "chat.chat_ai.result", nil, err.Error(), http.StatusInternalServerError)
	}

	vars := make(model.Variables)
	if argv.Response != "" {
		vars[argv.Response] = res.ResponseMessage
	}
	if argv.DefinedCategories != "" {
		category, _ := json.Marshal(res.UsedCategories)
		vars[argv.DefinedCategories] = string(category)
	}

	return conv.Set(ctx, vars)
}
