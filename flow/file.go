package flow

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/webitel/flow_manager/model"
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

type PrintFileArgs struct {
	Files []model.File `json:"files"`
}

func (r *router) printFile(ctx context.Context, scope *Flow, conn model.Connection, args interface{}) (model.Response, *model.AppError) {
	var argv PrintFileArgs
	err := scope.Decode(args, &argv)
	if err != nil {
		return nil, err
	}

	fmt.Println(argv)

	return ResponseOK, nil
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

	parsedId, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		return model.CallResponseError, ErrorRequiredParameter("GenerateLink", "file id")
	}

	link, appErr := r.fm.GeneratePreSignedResourceSignature(ctx, "download", "file", parsedId, conn.DomainId(), map[string]string{"expire": strconv.FormatInt(argv.Expire*1000, 10)})
	if appErr != nil {
		return nil, appErr
	}

	if argv.Source != "" {
		link += "&source=" + argv.Source
	}

	return conn.Set(ctx, model.Variables{
		argv.Set: server + link,
	})
}
