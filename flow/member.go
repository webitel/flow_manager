package flow

import (
	"context"
	"github.com/webitel/flow_manager/model"
)

type queuePosition struct {
	Set string `json:"set"`
}

type GetMemberInfo struct {
	Member *model.SearchMember `json:"member"`
	Set    model.Variables
}

func (r *router) QueueCallPosition(ctx context.Context, scope *Flow, call model.Connection, args interface{}) (model.Response, *model.AppError) {
	var argv queuePosition

	if err := scope.Decode(args, &argv); err != nil {
		return nil, err
	}

	if argv.Set == "" {
		return nil, ErrorRequiredParameter("queueCallPosition", "SET")
	}

	pos, err := r.fm.GetCallPosition(call.Id())
	if err != nil {
		return nil, err
	}

	return call.Set(ctx, model.Variables{
		argv.Set: pos,
	})
}

func (r *router) GetMember(ctx context.Context, scope *Flow, conn model.Connection, args interface{}) (model.Response, *model.AppError) {
	var argv GetMemberInfo
	var err *model.AppError
	if err = scope.Decode(args, &argv); err != nil {
		return nil, err
	}

	if argv.Member == nil {
		return nil, ErrorRequiredParameter("GetMember", "member")
	}

	if argv.Set == nil {
		return nil, ErrorRequiredParameter("GetMember", "set")
	}

	res, err := r.fm.GetMemberProperties(conn.DomainId(), argv.Member, argv.Set)
	if err != nil {
		return nil, err
	}

	return conn.Set(ctx, res)
}
