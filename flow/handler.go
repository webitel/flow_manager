package flow

import (
	"context"
	"fmt"
	"github.com/webitel/flow_manager/model"
	"github.com/webitel/wlog"
)

type Handler interface {
	Request(ctx *Flow, req model.ApplicationRequest) (model.Response, *model.AppError)
}

func Route(ctx context.Context, i *Flow, handler Handler) {
	var req *ApplicationRequest
	var err *model.AppError
	var res model.Response

	wlog.Debug(fmt.Sprintf("flow \"%s\" start conn %s", i.name, i.Connection.Id()))
	defer wlog.Debug(fmt.Sprintf("flow \"%s\" stopped conn %s", i.name, i.Connection.Id()))

	for {
		req = i.NextRequest()
		if req == nil {
			break
		}

		if res, err = handler.Request(i, req); err != nil {
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
