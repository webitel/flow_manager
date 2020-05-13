package flow

import (
	"context"
	"github.com/webitel/flow_manager/model"
	"github.com/webitel/wlog"
)

type LogArg string

func (r *router) Log(ctx context.Context, scope *Flow, args interface{}) model.ResultChannel {
	return Do(func(result *model.Result) {
		var log LogArg
		if err := Decode(scope.Connection, args, &log); err != nil {
			result.Err = err
		} else {
			wlog.Info(string(log))
			result.Res = model.CallResponseOK
		}
	})
}
