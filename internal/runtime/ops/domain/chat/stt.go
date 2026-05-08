package chat

import (
	"context"
	"fmt"
	"net/http"

	"github.com/webitel/flow_manager/internal/runtime/ops"
	"github.com/webitel/flow_manager/model"
)

// STTDeps is the narrow interface required by the stt op.
type STTDeps interface {
	GetFileTranscription(ctx context.Context, fileId, domainId int64, profileId int64, language string) (string, *model.AppError)
}

// RegisterSTT adds the stt op to reg.
func RegisterSTT(reg *ops.Registry, deps STTDeps) {
	reg.Register("stt", &sttOp{deps: deps})
}

type sttOp struct{ deps STTDeps }

func (o *sttOp) Kind() ops.OpKind { return ops.OpKindSync }

type sttArgs struct {
	FileId    int64  `json:"fileId,omitempty"`
	ProfileId int64  `json:"profileId,omitempty"`
	Language  string `json:"language,omitempty"`
	SetVar    string `json:"setVar,omitempty"`
}

func (o *sttOp) Execute(ctx context.Context, in ops.OpInput) (ops.OpOutput, error) {
	conv, ok := conversationFromContext(ctx)
	if !ok {
		return ops.OpOutput{}, fmt.Errorf("stt: no conversation in context")
	}
	var argv sttArgs
	if err := ops.DecodeArgs(in, &argv); err != nil {
		return ops.OpOutput{}, err
	}
	if argv.FileId <= 0 {
		return ops.OpOutput{}, model.NewAppError("stt", "chat_route.stt.check_args.error", nil, "file id invalid", http.StatusBadRequest)
	}
	if argv.ProfileId <= 0 {
		return ops.OpOutput{}, model.NewAppError("stt", "chat_route.stt.check_args.error", nil, "profile id invalid", http.StatusBadRequest)
	}
	if argv.Language == "" {
		return ops.OpOutput{}, model.NewAppError("stt", "chat_route.stt.check_args.error", nil, "language empty", http.StatusBadRequest)
	}
	if argv.SetVar == "" {
		return ops.OpOutput{}, model.NewAppError("stt", "chat_route.stt.check_args.error", nil, "set var empty", http.StatusBadRequest)
	}

	text, appErr := o.deps.GetFileTranscription(ctx, argv.FileId, in.DomainID, argv.ProfileId, argv.Language)
	if appErr != nil {
		return ops.OpOutput{}, fmt.Errorf("stt: %s", appErr.Error())
	}

	if _, appErr := conv.Set(ctx, model.Variables{argv.SetVar: text}); appErr != nil {
		return ops.OpOutput{}, fmt.Errorf("stt: set var: %s", appErr.Error())
	}
	return ops.OpOutput{SetVars: map[string]string{argv.SetVar: text}}, nil
}
