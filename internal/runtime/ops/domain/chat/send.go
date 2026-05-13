package chat

import (
	"context"
	"fmt"
	"net/url"

	chatdomain "github.com/webitel/flow_manager/internal/domain/chat"
	"github.com/webitel/flow_manager/internal/domain/files"
	"github.com/webitel/flow_manager/internal/runtime/ops"
)

// SendDeps is the subset of  the send ops need.
type SendDeps interface {
	SearchMediaFile(domainId int64, search *files.SearchFile) (*files.File, error)
	SetupPublicFileUrl(file *files.File, domainId int64, server, source string, expire int64) (*files.File, error)
	SenChatAction(ctx context.Context, channelId string, action chatdomain.ChatAction) error
	GenerateTTSLink(ctx context.Context, text string, domainId int64, profileId int, textType string, voice string, language string) (string, error)
}

// RegisterSend registers sendMessage, sendText, sendFile, sendImage, sendAction, sendTts.
func RegisterSend(reg *ops.Registry, deps SendDeps) {
	reg.Register("sendMessage", &sendMessageOp{deps: deps})
	reg.Register("sendText", &sendTextOp{})
	reg.Register("sendFile", &sendFileOp{deps: deps})
	reg.Register("sendImage", &sendImageOp{})
	reg.Register("sendAction", &sendActionOp{deps: deps})
	reg.Register("sendTts", &sendTtsOp{deps: deps})
}

// ── sendMessage ───────────────────────────────────────────────────────────────

type sendMessageOp struct{ deps SendDeps }

func (o *sendMessageOp) Kind() ops.OpKind { return ops.OpKindSync }

func (o *sendMessageOp) Execute(ctx context.Context, in ops.OpInput) (ops.OpOutput, error) {
	conv, ok := conversationFromContext(ctx)
	if !ok {
		return ops.OpOutput{}, fmt.Errorf("sendMessage: no conversation in context")
	}
	var argv chatdomain.ChatMessageOutbound
	if err := ops.DecodeArgs(in, &argv); err != nil {
		return ops.OpOutput{}, err
	}
	if argv.File != nil {
		if argv.File.Url != "" && argv.File.Id == 0 {
			argv.File.Id = 1
		} else {
			server := resolveServer(argv.File.Server, argv.Server)
			var appErr error
			argv.File, appErr = o.deps.SearchMediaFile(in.DomainID, &files.SearchFile{
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
			argv.File.MimeType += ";source=media"
			argv.Type = "file"
		}
	}
	if _, appErr := conv.SendMessage(ctx, argv); appErr != nil {
		return ops.OpOutput{}, fmt.Errorf("sendMessage: %s", appErr.Error())
	}
	return ops.OpOutput{}, nil
}

// ── sendText ──────────────────────────────────────────────────────────────────

type sendTextOp struct{}

func (sendTextOp) Kind() ops.OpKind { return ops.OpKindSync }

func (sendTextOp) Execute(ctx context.Context, in ops.OpInput) (ops.OpOutput, error) {
	conv, ok := conversationFromContext(ctx)
	if !ok {
		return ops.OpOutput{}, fmt.Errorf("sendText: no conversation in context")
	}
	text, _ := in.Node.RawArgs.(string)
	text = ops.ExpandStr(text, in.Variables, in.GlobalVar)
	if _, appErr := conv.SendTextMessage(ctx, text); appErr != nil {
		return ops.OpOutput{}, fmt.Errorf("sendText: %s", appErr.Error())
	}
	return ops.OpOutput{}, nil
}

// ── sendFile ──────────────────────────────────────────────────────────────────

type sendFileOp struct{ deps SendDeps }

func (o *sendFileOp) Kind() ops.OpKind { return ops.OpKindSync }

type sendFileArgs struct {
	Id     int              `json:"id"` // deprecated alias for file.id
	File   files.SearchFile `json:"file"`
	Text   string           `json:"text"`
	Source string           `json:"source"`
	Expire int64            `json:"expire"`
	Server string           `json:"server"`
	Kind   string           `json:"kind"`
}

func (o *sendFileOp) Execute(ctx context.Context, in ops.OpInput) (ops.OpOutput, error) {
	conv, ok := conversationFromContext(ctx)
	if !ok {
		return ops.OpOutput{}, fmt.Errorf("sendFile: no conversation in context")
	}
	var argv sendFileArgs
	if err := ops.DecodeArgs(in, &argv); err != nil {
		return ops.OpOutput{}, err
	}
	if argv.Id > 0 && argv.File.Id == 0 {
		argv.File.Id = argv.Id
	}
	server := resolveServer(argv.Server, "")
	file, appErr := o.deps.SearchMediaFile(in.DomainID, &argv.File)
	if appErr != nil {
		return ops.OpOutput{}, fmt.Errorf("sendFile: search media: %s", appErr.Error())
	}
	if file == nil {
		return ops.OpOutput{}, fmt.Errorf("sendFile: file not found")
	}
	if argv.Expire == 0 {
		argv.Expire = 604800
	}
	file, appErr = o.deps.SetupPublicFileUrl(file, in.DomainID, server, argv.Source, argv.Expire)
	if appErr != nil {
		return ops.OpOutput{}, fmt.Errorf("sendFile: setup url: %s", appErr.Error())
	}
	file.MimeType += ";source=media"
	if _, appErr := conv.SendFile(ctx, argv.Text, file, argv.Kind); appErr != nil {
		return ops.OpOutput{}, fmt.Errorf("sendFile: %s", appErr.Error())
	}
	return ops.OpOutput{}, nil
}

// ── sendImage ─────────────────────────────────────────────────────────────────

type sendImageOp struct{}

func (sendImageOp) Kind() ops.OpKind { return ops.OpKindSync }

type sendImageArgs struct {
	Url  string `json:"url"`
	Name string `json:"name"`
	Text string `json:"text"`
	Kind string `json:"kind"`
}

func (sendImageOp) Execute(ctx context.Context, in ops.OpInput) (ops.OpOutput, error) {
	conv, ok := conversationFromContext(ctx)
	if !ok {
		return ops.OpOutput{}, fmt.Errorf("sendImage: no conversation in context")
	}
	argv := sendImageArgs{Name: "empty"}
	if err := ops.DecodeArgs(in, &argv); err != nil {
		return ops.OpOutput{}, err
	}
	u, e := url.ParseRequestURI(argv.Url)
	if e != nil {
		return ops.OpOutput{}, fmt.Errorf("sendImage: invalid url: %s", e.Error())
	}
	u.RawQuery = u.Query().Encode()
	if _, appErr := conv.SendImageMessage(ctx, u.String(), argv.Name, argv.Text, argv.Kind); appErr != nil {
		return ops.OpOutput{}, fmt.Errorf("sendImage: %s", appErr.Error())
	}
	return ops.OpOutput{}, nil
}

// ── sendAction ────────────────────────────────────────────────────────────────

type sendActionOp struct{ deps SendDeps }

func (o *sendActionOp) Kind() ops.OpKind { return ops.OpKindSync }

type sendActionArgs struct {
	Action chatdomain.ChatAction `json:"action"`
}

func (o *sendActionOp) Execute(ctx context.Context, in ops.OpInput) (ops.OpOutput, error) {
	conv, ok := conversationFromContext(ctx)
	if !ok {
		return ops.OpOutput{}, fmt.Errorf("sendAction: no conversation in context")
	}
	var argv sendActionArgs
	if err := ops.DecodeArgs(in, &argv); err != nil {
		return ops.OpOutput{}, err
	}
	if appErr := o.deps.SenChatAction(ctx, conv.Id(), argv.Action); appErr != nil {
		return ops.OpOutput{}, fmt.Errorf("sendAction: %s", appErr.Error())
	}
	return ops.OpOutput{}, nil
}

// ── sendTts ───────────────────────────────────────────────────────────────────

type sendTtsOp struct{ deps SendDeps }

func (o *sendTtsOp) Kind() ops.OpKind { return ops.OpKindSync }

type sendTtsArgs struct {
	Message   string `json:"message,omitempty"`
	ProfileId int    `json:"profileId,omitempty"`
	Language  string `json:"language,omitempty"`
	Voice     string `json:"voice,omitempty"`
	TextType  string `json:"textType,omitempty"`
	Server    string `json:"server,omitempty"`
	FileName  string `json:"fileName,omitempty"`
	Kind      string `json:"kind,omitempty"`
}

func (o *sendTtsOp) Execute(ctx context.Context, in ops.OpInput) (ops.OpOutput, error) {
	conv, ok := conversationFromContext(ctx)
	if !ok {
		return ops.OpOutput{}, fmt.Errorf("sendTts: no conversation in context")
	}
	var argv sendTtsArgs
	if err := ops.DecodeArgs(in, &argv); err != nil {
		return ops.OpOutput{}, err
	}
	uri, appErr := o.deps.GenerateTTSLink(ctx, argv.Message, in.DomainID, argv.ProfileId, argv.TextType, argv.Voice, argv.Language)
	if appErr != nil {
		return ops.OpOutput{}, fmt.Errorf("sendTts: %s", appErr.Error())
	}
	file := &files.File{
		Url:      argv.Server + uri,
		MimeType: "audio/mpeg",
		Name:     argv.FileName,
		Id:       -1,
	}
	if _, appErr := conv.SendFile(ctx, "", file, argv.Kind); appErr != nil {
		return ops.OpOutput{}, fmt.Errorf("sendTts: send file: %s", appErr.Error())
	}
	return ops.OpOutput{}, nil
}
