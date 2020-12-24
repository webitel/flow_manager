package flow

import (
	"context"
	"github.com/webitel/flow_manager/model"
	"strings"
)

type GenerateLinkArgs struct {
	Server  string `json:"server"`
	Timeout int64  `json:"timeout"` // sec ?
	Set     string `json:"set"`
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

	link, err := r.fm.GeneratePreSignetResourceSignature("/any/file", "download", conn.Id(), conn.DomainId(), argv.Timeout*1000)
	if err != nil {
		return nil, err
	}

	return conn.Set(ctx, model.Variables{
		argv.Set: server + link,
	})
}
