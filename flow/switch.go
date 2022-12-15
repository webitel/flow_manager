package flow

import (
	"context"
	"fmt"
	"net/http"

	"github.com/webitel/flow_manager/model"
	"github.com/webitel/wlog"
)

func (r *router) switchHandler(ctx context.Context, scope *Flow, conn model.Connection, args interface{}) (model.Response, *model.AppError) {
	var req *SwitchVal
	var ok bool
	var key TreeKey = -1

	if req, ok = args.(*SwitchVal); !ok {
		return nil, model.NewAppError("Flow.SwitchHandler", "flow.condition_switch.not_found", nil, "bad arguments", http.StatusBadRequest)
	}

	if key, ok = req.Cases[conn.ParseText(req.Variable)]; ok {
		scope.tree.Current = key
		wlog.Debug(fmt.Sprintf("[%s] set switch case: %s", conn.Id(), req.Variable))
	} else if key, ok = req.Cases["default"]; ok {
		scope.tree.Current = key
		wlog.Debug(fmt.Sprintf("call %s set switch default case %s", conn.Id(), req.Variable))
	}

	return ResponseOK, nil
}
