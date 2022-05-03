package processing

import (
	"context"
	"strconv"

	"github.com/webitel/flow_manager/flow"
	"github.com/webitel/flow_manager/model"
)

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
