package flow

import (
	"context"
	"fmt"
	"github.com/webitel/flow_manager/model"
	"github.com/webitel/wlog"
)

type queuePosition struct {
	Set string `json:"set"`
}

type GetMemberInfo struct {
	Member *model.SearchMember `json:"member"`
	Set    model.Variables
}

type UpdateMembers struct {
	Member *model.SearchMember `json:"member"`
	Patch  *model.PatchMember  `json:"patch"`
}

type EWTCall struct {
	SetVar    string `json:"setVar"`
	QueueIds  []int  `json:"queue_ids"`
	BucketIds []int  `json:"bucket_ids"`

	Strategy string `json:"strategy"`
	Min      int    `json:"min"`
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

func (r *router) PatchMembers(ctx context.Context, scope *Flow, conn model.Connection, args interface{}) (model.Response, *model.AppError) {
	var argv UpdateMembers
	var err *model.AppError
	if err = scope.Decode(args, &argv); err != nil {
		return nil, err
	}

	if argv.Member == nil {
		return nil, ErrorRequiredParameter("PatchMembers", "member")
	}

	if argv.Patch == nil {
		return nil, ErrorRequiredParameter("PatchMembers", "patch")
	}

	res, err := r.fm.PatchMembers(conn.DomainId(), argv.Member, argv.Patch)
	if err != nil {
		return nil, err
	}

	if res > 0 {
		wlog.Debug(fmt.Sprintf("[%s] patch members count %d", conn.Id(), res))
	}

	return model.CallResponseOK, nil
}

func (r *router) EWTCall(ctx context.Context, scope *Flow, conn model.Connection, args interface{}) (model.Response, *model.AppError) {
	var argv EWTCall
	var err *model.AppError
	var ewt float64
	if err = scope.Decode(args, &argv); err != nil {
		return nil, err
	}

	if argv.SetVar == "" {
		return nil, ErrorRequiredParameter("EWTCall", "setVar")
	}

	if argv.Min == 0 {
		argv.Min = 60
	}

	ewt, err = r.fm.EWTPuzzle(conn.DomainId(), conn.Id(), argv.Min, argv.QueueIds, argv.BucketIds)
	if err != nil {
		return model.CallResponseError, err
	}

	return conn.Set(ctx, model.Variables{
		argv.SetVar: fmt.Sprintf("%f", ewt),
	})
}
