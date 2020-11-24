package flow

import (
	"context"
	"github.com/webitel/flow_manager/model"
)

type GetUserInfo struct {
	User *model.SearchUser `json:"user"`
	Set  model.Variables
}

func (r *router) GetUser(ctx context.Context, scope *Flow, c model.Connection, args interface{}) (model.Response, *model.AppError) {
	var argv GetUserInfo
	var err *model.AppError
	if err = scope.Decode(args, &argv); err != nil {
		return nil, err
	}

	if argv.User == nil {
		return nil, ErrorRequiredParameter("GetUser", "user")
	}

	if argv.Set == nil {
		return nil, ErrorRequiredParameter("GetUser", "set")
	}

	res, err := r.fm.GetUserProperties(c.DomainId(), argv.User, argv.Set)
	if err != nil {
		return nil, err
	}

	return c.Set(ctx, res)
}
