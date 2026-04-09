package im

import (
	"context"
	"fmt"
	"net/http"

	"github.com/webitel/flow_manager/flow"
	"github.com/webitel/flow_manager/model"
)

type TTSArgs struct {
	Message   string `json:"message,omitempty"`
	ProfileID int    `json:"profileId,omitempty"`
	Language  string `json:"language,omitempty"`
	Voice     string `json:"voice,omitempty"`
	TextType  string `json:"textType,omitempty"`
	Server    string `json:"server,omitempty"`
	FileName  string `json:"fileName,omitempty"`
	Kind      string `json:"kind,omitempty"`
}

func (r *Router) sendTTS(ctx context.Context, scope *flow.Flow, conv Dialog, args any) (model.Response, *model.AppError) {
	var argv TTSArgs
	var err *model.AppError

	if err = r.Decode(scope, args, &argv); err != nil {
		return nil, err
	}

	uri, err := r.fm.GenerateTTSLink(ctx, argv.Message, conv.DomainId(), argv.ProfileID, argv.TextType, argv.Voice, argv.Language)
	if err != nil {
		return model.CallResponseError, err
	}
	fmt.Println(uri)
	// TODO: mime type
	return conv.SendFile(ctx, "", &model.File{Url: argv.Server + uri, MimeType: "audio/mpeg", Name: argv.FileName, Id: -1, Size: 1}, argv.Kind)
}

type STTArgs struct {
	FileID    int64  `json:"fileId,omitempty"`
	ProfileID int64  `json:"profileId,omitempty"`
	Language  string `json:"language,omitempty"`
	SetVar    string `json:"setVar,omitempty"`
}

func (r *Router) STT(ctx context.Context, scope *flow.Flow, conv Dialog, args any) (model.Response, *model.AppError) {
	var argv STTArgs
	var err *model.AppError

	if err = r.Decode(scope, args, &argv); err != nil {
		return nil, err
	}

	if argv.FileID <= 0 {
		return model.CallResponseError, model.NewAppError("", "im_route.stt.check_args.error", nil, "file id invalid", http.StatusBadRequest)
	}
	if argv.ProfileID <= 0 {
		return model.CallResponseError, model.NewAppError("", "im_route.stt.check_args.error", nil, "profile id invalid", http.StatusBadRequest)
	}
	if argv.Language == "" {
		return model.CallResponseError, model.NewAppError("", "im_route.stt.check_args.error", nil, "language empty", http.StatusBadRequest)
	}
	if argv.SetVar == "" {
		return model.CallResponseError, model.NewAppError("", "im_route.stt.check_args.error", nil, "set var empty", http.StatusBadRequest)
	}

	transcription, err := r.fm.GetFileTranscription(ctx, argv.FileID, conv.DomainId(), argv.ProfileID, argv.Language)
	if err != nil {
		return model.CallResponseError, err
	}
	return conv.Set(ctx, model.Variables{
		argv.SetVar: transcription,
	})
}
