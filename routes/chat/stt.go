package chat

import (
	"context"
	"net/http"

	"github.com/webitel/flow_manager/flow"
	"github.com/webitel/flow_manager/model"
)

type STTArgs struct {
	FileId    int64  `json:"fileId,omitempty"`
	ProfileId int64  `json:"profileId,omitempty"`
	Language  string `json:"language,omitempty"`
	SetVar    string `json:"setVar,omitempty"`
}

func (r *Router) STT(ctx context.Context, scope *flow.Flow, conv Conversation, args any) (model.Response, *model.AppError) {
	var argv STTArgs
	var err *model.AppError

	if err = r.Decode(scope, args, &argv); err != nil {
		return nil, err
	}

	if argv.FileId <= 0 {
		return model.CallResponseError, model.NewAppError("", "chat_route.stt.check_args.error", nil, "file id invalid", http.StatusBadRequest)
	}
	if argv.ProfileId <= 0 {
		return model.CallResponseError, model.NewAppError("", "chat_route.stt.check_args.error", nil, "profile id invalid", http.StatusBadRequest)
	}
	if argv.Language == "" {
		return model.CallResponseError, model.NewAppError("", "chat_route.stt.check_args.error", nil, "language empty", http.StatusBadRequest)
	}
	if argv.SetVar == "" {
		return model.CallResponseError, model.NewAppError("", "chat_route.stt.check_args.error", nil, "set var empty", http.StatusBadRequest)
	}

	transcription, err := r.fm.GetFileTranscription(ctx, argv.FileId, conv.DomainId(), argv.ProfileId, argv.Language)
	if err != nil {
		return model.CallResponseError, err
	}
	return conv.Set(ctx, model.Variables{
		argv.SetVar: transcription,
	})
}
