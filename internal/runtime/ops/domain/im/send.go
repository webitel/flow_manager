package im

import (
	"context"
	"fmt"
	"strings"

	"github.com/webitel/flow_manager/internal/runtime/ops"
	"github.com/webitel/flow_manager/model"
)

// SendDeps is the subset of ports.RouterDeps that the send ops need.
type SendDeps interface {
	SearchMediaFile(domainId int64, search *model.SearchFile) (*model.File, *model.AppError)
	SetupPublicFileUrl(file *model.File, domainId int64, server, source string, expire int64) (*model.File, *model.AppError)
	SenChatAction(ctx context.Context, channelId string, action model.ChatAction) *model.AppError
}

// RegisterSend registers sendMessage, sendText, sendImage, sendFile, sendAction.
func RegisterSend(reg *ops.Registry, deps SendDeps) {
	reg.Register("sendMessage", &sendMessageOp{deps: deps})
	reg.Register("sendText", &sendTextOp{})
	reg.Register("sendImage", &sendImageOp{deps: deps})
	reg.Register("sendFile", &sendFileOp{deps: deps})
	reg.Register("sendAction", &sendActionOp{deps: deps})
}

// ── sendMessage ───────────────────────────────────────────────────────────────

type sendMessageOp struct{ deps SendDeps }

func (o *sendMessageOp) Kind() ops.OpKind { return ops.OpKindSync }

func (o *sendMessageOp) Execute(ctx context.Context, in ops.OpInput) (ops.OpOutput, error) {
	dialog, ok := dialogFromContext(ctx)
	if !ok {
		return ops.OpOutput{}, fmt.Errorf("sendMessage: no IMDialog in context")
	}

	var argv model.ChatMessageOutbound
	if err := ops.DecodeArgs(in, &argv); err != nil {
		return ops.OpOutput{}, err
	}

	if argv.File != nil {
		if argv.File.Url != "" && argv.File.Id == 0 {
			argv.File.Id = 1
		} else {
			server := resolveServer(argv.File.Server, argv.Server)
			var appErr *model.AppError
			argv.File, appErr = o.deps.SearchMediaFile(in.DomainID, &model.SearchFile{
				Id:   argv.File.Id,
				Name: argv.File.Name,
			})
			if appErr != nil {
				return ops.OpOutput{}, fmt.Errorf("sendMessage: search media: %s", appErr.Error())
			}
			argv.File, appErr = o.deps.SetupPublicFileUrl(argv.File, in.DomainID, server, "media", 604800)
			if appErr != nil {
				return ops.OpOutput{}, fmt.Errorf("sendMessage: setup url: %s", appErr.Error())
			}
			argv.Type = "file"
		}
	}

	if _, appErr := dialog.SendMessage(ctx, argv); appErr != nil {
		return ops.OpOutput{}, fmt.Errorf("sendMessage: %s", appErr.Error())
	}
	return ops.OpOutput{}, nil
}

// ── sendText ──────────────────────────────────────────────────────────────────

type sendTextOp struct{}

func (o *sendTextOp) Kind() ops.OpKind { return ops.OpKindSync }

func (o *sendTextOp) Execute(ctx context.Context, in ops.OpInput) (ops.OpOutput, error) {
	dialog, ok := dialogFromContext(ctx)
	if !ok {
		return ops.OpOutput{}, fmt.Errorf("sendText: no IMDialog in context")
	}

	text, _ := in.Node.RawArgs.(string)
	text = ops.ExpandStr(text, in.Variables, in.GlobalVar)

	if _, appErr := dialog.SendTextMessage(ctx, text); appErr != nil {
		return ops.OpOutput{}, fmt.Errorf("sendText: %s", appErr.Error())
	}
	return ops.OpOutput{}, nil
}

// ── sendImage ─────────────────────────────────────────────────────────────────

type sendImageOp struct{ deps SendDeps }

func (o *sendImageOp) Kind() ops.OpKind { return ops.OpKindSync }

func (o *sendImageOp) Execute(ctx context.Context, in ops.OpInput) (ops.OpOutput, error) {
	dialog, ok := dialogFromContext(ctx)
	if !ok {
		return ops.OpOutput{}, fmt.Errorf("sendImage: no IMDialog in context")
	}

	var argv model.ChatMessageOutbound
	if err := ops.DecodeArgs(in, &argv); err != nil {
		return ops.OpOutput{}, err
	}

	if argv.File != nil {
		if argv.File.Url != "" && argv.File.Id == 0 {
			argv.File.Id = 1
		} else {
			server := resolveServer(argv.File.Server, argv.Server)
			var appErr *model.AppError
			argv.File, appErr = o.deps.SearchMediaFile(in.DomainID, &model.SearchFile{
				Id:   argv.File.Id,
				Name: argv.File.Name,
			})
			if appErr != nil {
				return ops.OpOutput{}, fmt.Errorf("sendImage: search media: %s", appErr.Error())
			}
			argv.File, appErr = o.deps.SetupPublicFileUrl(argv.File, in.DomainID, server, "media", 604800)
			if appErr != nil {
				return ops.OpOutput{}, fmt.Errorf("sendImage: setup url: %s", appErr.Error())
			}
		}
	}

	if _, appErr := dialog.SendImageMessage(ctx, argv); appErr != nil {
		return ops.OpOutput{}, fmt.Errorf("sendImage: %s", appErr.Error())
	}
	return ops.OpOutput{}, nil
}

// ── sendFile ──────────────────────────────────────────────────────────────────

type sendFileOp struct{ deps SendDeps }

func (o *sendFileOp) Kind() ops.OpKind { return ops.OpKindSync }

func (o *sendFileOp) Execute(ctx context.Context, in ops.OpInput) (ops.OpOutput, error) {
	dialog, ok := dialogFromContext(ctx)
	if !ok {
		return ops.OpOutput{}, fmt.Errorf("sendFile: no IMDialog in context")
	}

	var argv model.ChatMessageOutbound
	if err := ops.DecodeArgs(in, &argv); err != nil {
		return ops.OpOutput{}, err
	}

	if argv.File != nil {
		if argv.File.Url != "" && argv.File.Id == 0 {
			argv.File.Id = 1
		} else {
			server := resolveServer(argv.File.Server, argv.Server)
			var appErr *model.AppError
			argv.File, appErr = o.deps.SearchMediaFile(in.DomainID, &model.SearchFile{
				Id:   argv.File.Id,
				Name: argv.File.Name,
			})
			if appErr != nil {
				return ops.OpOutput{}, fmt.Errorf("sendFile: search media: %s", appErr.Error())
			}
			argv.File, appErr = o.deps.SetupPublicFileUrl(argv.File, in.DomainID, server, "media", 604800)
			if appErr != nil {
				return ops.OpOutput{}, fmt.Errorf("sendFile: setup url: %s", appErr.Error())
			}
		}
	}

	if _, appErr := dialog.SendDocumentMessage(ctx, argv); appErr != nil {
		return ops.OpOutput{}, fmt.Errorf("sendFile: %s", appErr.Error())
	}
	return ops.OpOutput{}, nil
}

// ── sendAction ────────────────────────────────────────────────────────────────

type sendActionOp struct{ deps SendDeps }

func (o *sendActionOp) Kind() ops.OpKind { return ops.OpKindSync }

type sendActionArgs struct {
	Action model.ChatAction `json:"action"`
}

func (o *sendActionOp) Execute(ctx context.Context, in ops.OpInput) (ops.OpOutput, error) {
	dialog, ok := dialogFromContext(ctx)
	if !ok {
		return ops.OpOutput{}, fmt.Errorf("sendAction: no IMDialog in context")
	}

	var argv sendActionArgs
	if err := ops.DecodeArgs(in, &argv); err != nil {
		return ops.OpOutput{}, err
	}

	if appErr := o.deps.SenChatAction(ctx, dialog.Id(), argv.Action); appErr != nil {
		return ops.OpOutput{}, fmt.Errorf("sendAction: %s", appErr.Error())
	}
	return ops.OpOutput{}, nil
}

// resolveServer picks a non-empty server string and strips a trailing slash.
func resolveServer(fileServer, fallback string) string {
	s := fileServer
	if s == "" {
		s = fallback
	}
	return strings.TrimSuffix(s, "/")
}

// rawStringSlice extracts a []string from in.Node.RawArgs, expanding variables.
// Used for ops whose schema args are a JSON array: ["var1", "var2"].
func rawStringSlice(in ops.OpInput) []string {
	switch v := in.Node.RawArgs.(type) {
	case []any:
		out := make([]string, 0, len(v))
		for _, item := range v {
			if s, ok := item.(string); ok {
				out = append(out, ops.ExpandStr(s, in.Variables, in.GlobalVar))
			}
		}
		return out
	case string:
		if s := ops.ExpandStr(v, in.Variables, in.GlobalVar); s != "" {
			return []string{s}
		}
	}
	return nil
}
