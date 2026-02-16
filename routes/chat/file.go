package chat

import (
	"context"
	"net/http"
	"net/url"
	"strings"

	"github.com/webitel/flow_manager/flow"
	"github.com/webitel/flow_manager/model"
)

type FileMessage struct {
	Id     int              `json:"id"` // todo deprecated
	File   model.SearchFile `json:"file"`
	Text   string           `json:"text"`
	Source string           `json:"source"`
	Expire int64            `json:"expire"`
	Server string           `json:"server"`
	Url    string
	Kind   string `json:"kind"`
}

type ImageArgv struct {
	Url  string
	Name string
	Text string
	Kind string
}

func (r *Router) sendImage(ctx context.Context, scope *flow.Flow, conv Conversation, args any) (model.Response, *model.AppError) {
	var err *model.AppError
	argv := ImageArgv{
		Name: "empty",
	}

	if err = r.Decode(scope, args, &argv); err != nil {
		return nil, err
	}

	u, e := url.ParseRequestURI(argv.Url)
	if e != nil {
		return model.CallResponseError, model.NewAppError("sendImage", "chat.send_image.valid.url", nil, "bad arguments", http.StatusBadRequest)
	}

	u.RawQuery = u.Query().Encode()

	return conv.SendImageMessage(ctx, u.String(), argv.Name, argv.Text, argv.Kind)
}

func (r *Router) sendFile(ctx context.Context, scope *flow.Flow, conv Conversation, args any) (model.Response, *model.AppError) {
	var argv FileMessage
	var file *model.File
	var err *model.AppError

	if err = r.Decode(scope, args, &argv); err != nil {
		return nil, err
	}

	if argv.Id > 0 && argv.File.Id == 0 {
		argv.File.Id = argv.Id
	}

	server := argv.Server

	if strings.HasSuffix(server, "/") {
		server = server[:len(server)-1]
	}

	switch argv.Source {
	default:
		if file, err = r.fm.SearchMediaFile(conv.DomainId(), &argv.File); err != nil {
			return nil, err
		}
	}

	if file == nil {
		return model.CallResponseError, nil
	}

	if argv.Expire == 0 {
		argv.Expire = 604800
	}

	file, err = r.fm.SetupPublicFileUrl(file, conv.DomainId(), server, argv.Source, argv.Expire)
	if err != nil {
		return nil, err
	}

	file.MimeType += ";source=media"

	return conv.SendFile(ctx, argv.Text, file, argv.Kind)
}
