package flow

import (
	"context"
	"github.com/webitel/flow_manager/model"
)

type MeetingArgs struct {
	SetVar    string            `json:"setVar"`
	Title     string            `json:"title,omitempty"`
	ExpireSec int64             `json:"expireSec,omitempty"`
	BasePath  string            `json:"basePath,omitempty"`
	Variables map[string]string `json:"variables,omitempty"`
}

func (r *router) createMeeting(ctx context.Context, scope *Flow, conn model.Connection, args any) (model.Response, *model.AppError) {

	var (
		argv MeetingArgs
	)
	if err := scope.Decode(args, &argv); err != nil {
		return model.CallResponseError, err
	}

	if argv.SetVar == "" {
		return model.CallResponseError, ErrorRequiredParameter("createMeeting", "setVar")
	}

	url, err := r.fm.Meeting().CreateMeeting(ctx, conn.DomainId(), argv.Title, int(argv.ExpireSec), argv.BasePath, argv.Variables)
	if err != nil {
		return model.CallResponseError, err
	}

	return conn.Set(ctx, model.Variables{
		argv.SetVar: url,
	})
}
