package flow

import (
	"context"
	"fmt"
	"github.com/webitel/flow_manager/model"
	"github.com/webitel/wlog"
)

type Handler interface {
	Request(ctx context.Context, scope *Flow, req model.ApplicationRequest) <-chan model.Result
}

func Do(f func(result *model.Result)) model.ResultChannel {
	storeChannel := make(model.ResultChannel, 1)
	go func() {
		result := model.Result{}
		f(&result)
		storeChannel <- result
		close(storeChannel)
	}()
	return storeChannel
}

func Route(ctx context.Context, i *Flow, handler Handler) {
	var req *ApplicationRequest

	wlog.Debug(fmt.Sprintf("flow \"%s\" start conn %s", i.name, i.Connection.Id()))
	defer wlog.Debug(fmt.Sprintf("flow \"%s\" stopped conn %s", i.name, i.Connection.Id()))

	for {
		req = i.NextRequest()
		if req == nil {
			return
		}

		select {
		case <-ctx.Done():
			return
		case res := <-handler.Request(ctx, i, req):
			if res.Err != nil {
				wlog.Error(fmt.Sprintf("%v [%v] - %s", req.Id(), req.Args(), res.Err.Error()))
			} else {
				wlog.Debug(fmt.Sprintf("%v [%v] - %s", req.Id(), req.Args(), res.Res.String()))
			}

			if i.IsCancel() || req.IsCancel() {
				wlog.Debug(fmt.Sprintf("flow [%s] break", i.Name()))
				return
			}
		}
	}
}
