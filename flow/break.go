package flow

import (
	"context"
	"fmt"
	"github.com/webitel/flow_manager/model"
	"net/http"
)

type BreakArgs struct {
	Flow *Flow
}

func (r *router) breakHandler(ctx context.Context, scope *Flow, conn model.Connection, args interface{}) (model.Response, *model.AppError) {
	var req *BreakArgs
	var ok bool

	if req, ok = args.(*BreakArgs); ok {
		req.Flow.SetCancel()
		return model.CallResponseOK, nil
	}

	return nil, model.NewAppError("Flow.Break", "flow.app.break.valid.args", nil, fmt.Sprintf("bad arguments %v", args), http.StatusBadRequest)
}
