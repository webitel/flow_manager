package chat_route

import (
	"context"
	"github.com/webitel/flow_manager/flow"
	"github.com/webitel/flow_manager/model"
	"strconv"
	"strings"
)

type FileMessage struct {
	Id     int              `json:"id"` //todo deprecated
	File   model.SearchFile `json:"file"`
	Text   string           `json:"text"`
	Source string           `json:"source"`
	Expire int64            `json:"expire"`
	Server string           `json:"server"`
	Url    string
}

func (r *Router) sendFile(ctx context.Context, scope *flow.Flow, conv Conversation, args interface{}) (model.Response, *model.AppError) {
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

	link, err := r.fm.GeneratePreSignetResourceSignature("/any/file", "download", strconv.Itoa(file.Id), conv.DomainId(), argv.Expire*1000)
	if err != nil {
		return nil, err
	}

	if argv.Source != "" {
		link += "&source=" + argv.Source
	}
	file.Url = server + link

	return conv.SendFile(ctx, argv.Text, file)
}
