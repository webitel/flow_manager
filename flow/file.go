package flow

import (
	"context"
	"github.com/webitel/flow_manager/model"
	"strings"
)

type GenerateLinkArgs struct {
	Server string `json:"server"`
	Expire int64  `json:"expire"` // sec ?
	Set    string `json:"set"`
	Source string `json:"source"`
	File   struct {
		Id   string `json:"id"`
		Name string `json:"name"` // TODO
	} `json:"file"`
}

func (r *router) generateLink(ctx context.Context, scope *Flow, conn model.Connection, args interface{}) (model.Response, *model.AppError) {
	var argv GenerateLinkArgs
	if err := scope.Decode(args, &argv); err != nil {
		return nil, err
	}

	if argv.Set == "" {
		return nil, ErrorRequiredParameter("GenerateLink", "set")
	}

	server := argv.Server

	if strings.HasSuffix(server, "/") {
		server = server[:len(server)-1]
	}
	id := conn.Id()
	if argv.File.Id != "" {
		id = argv.File.Id
	}

	link, err := r.fm.GeneratePreSignetResourceSignature("/any/file", "download", id, conn.DomainId(), argv.Expire*1000)
	if err != nil {
		return nil, err
	}

	if argv.Source != "" {
		link += "&source=" + argv.Source
	}

	return conn.Set(ctx, model.Variables{
		argv.Set: server + link,
	})
}
