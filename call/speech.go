package call

import (
	"context"
	"encoding/json"
	"github.com/webitel/flow_manager/flow"
	"github.com/webitel/flow_manager/model"
	"net/http"
)

func (r *Router) speechAi(ctx context.Context, scope *flow.Flow, call model.Call, args interface{}) (model.Response, *model.AppError) {
	var argv model.ChatAiAnswer

	if err := r.Decode(scope, args, &argv); err != nil {
		return nil, err
	}

	if argv.HistoryLength == 0 {
		argv.HistoryLength = 10
	}

	messages := call.SpeechMessages(argv.HistoryLength)
	for _, m := range messages {
		argv.Messages = append(argv.Messages, model.ChatMessage{
			Text: m.Question,
			User: "",
		})
		argv.Messages = append(argv.Messages, model.ChatMessage{
			Text: m.Answer,
			User: "client",
		})
	}

	res, err := r.fm.ChatAnswerAi(ctx, call.DomainId(), argv)
	if err != nil {
		return model.CallResponseError, model.NewAppError("speechAi", "call.speech_ai.result", nil, err.Error(), http.StatusInternalServerError)
	}

	vars := make(model.Variables)
	if argv.Response != "" {
		vars[argv.Response] = res.ResponseMessage
	}
	if argv.DefinedCategories != "" {
		category, _ := json.Marshal(res.UsedCategories)
		vars[argv.DefinedCategories] = string(category)
	}

	return call.Set(ctx, vars)
}
