package processing

import (
	"context"
	"github.com/webitel/wlog"
	"strconv"

	"github.com/webitel/flow_manager/flow"
	"github.com/webitel/flow_manager/model"
)

type ResumeAttemptArgs struct {
	Id int `json:"id"`
}

func (r *Router) attemptResult(ctx context.Context, scope *flow.Flow, conn Connection, args interface{}) (model.Response, *model.AppError) {
	var argv model.AttemptResult
	var attId int
	tmp, _ := conn.Get("attempt_id")
	attId, _ = strconv.Atoi(tmp)

	if err := r.Decode(scope, args, &argv); err != nil {
		return nil, err
	}
	argv.Id = int64(attId)

	if err := r.fm.AttemptResult(&argv); err != nil {
		return nil, err
	}

	return model.CallResponseOK, nil
}

func (r *Router) resumeAttempt(ctx context.Context, scope *flow.Flow, conn Connection, args interface{}) (model.Response, *model.AppError) {
	var argv ResumeAttemptArgs

	if err := r.Decode(scope, args, &argv); err != nil {
		return nil, err
	}

	if argv.Id == 0 {
		tmp, _ := conn.Get("attempt_id")
		argv.Id, _ = strconv.Atoi(tmp)
	}

	err2 := r.fm.ResumeAttempt(ctx, int64(argv.Id), conn.DomainId())
	if err2 != nil {
		wlog.Error(err2.Error())
	}

	return model.CallResponseOK, nil
}
