package flow

import (
	"context"
	"fmt"
	"github.com/webitel/flow_manager/app"
	"github.com/webitel/flow_manager/model"
	"github.com/webitel/wlog"
)

func Route(ctx context.Context, i *Flow, handler app.Handler) {
	var req *ApplicationRequest
	var err *model.AppError
	var res model.Response

	wlog.Debug(fmt.Sprintf("flow \"%s\" start conn %s", i.name, i.conn.Id()))
	defer wlog.Debug(fmt.Sprintf("flow \"%s\" stopped conn %s", i.name, i.conn.Id()))

	for {
		req = i.NextRequest()
		if req == nil {
			break
		}

		if res, err = handler.Request(ctx, i.conn, req); err != nil {
			wlog.Error(fmt.Sprintf("%v [%v] - %s", req.Id(), req.Args(), err.Error()))
		} else {
			wlog.Debug(fmt.Sprintf("%v [%v] - %s", req.Id(), req.Args(), res.String()))
		}

		if i.IsCancel() || req.IsCancel() {
			wlog.Debug(fmt.Sprintf("flow [%s] break", i.Name()))
			break
		}
	}
}
