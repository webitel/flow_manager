package flow

import (
	"context"
	"fmt"
	"github.com/webitel/flow_manager/model"
	"github.com/webitel/wlog"
	"time"
)

type Handler interface {
	Request(ctx context.Context, scope *Flow, req model.ApplicationRequest) <-chan model.Result
}

func Do(f func(result *model.Result)) model.ResultChannel {
	storeChannel := make(model.ResultChannel, 1) // FIXME CHANNEL

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
	var s time.Time

	wlog.Debug(fmt.Sprintf("flow \"%s\" start conn %s", i.name, i.Connection.Id()))
	defer wlog.Debug(fmt.Sprintf("flow \"%s\" stopped conn %s", i.name, i.Connection.Id()))

	for {
		req = i.NextRequest()
		if req == nil {
			return
		}

		if i.IsCancel() {
			wlog.Debug(fmt.Sprintf("flow \"%s\" break", i.Name()))
			return
		}

		if req.limiter != nil {
			if req.limiter.MaxCount() {
				wlog.Debug(fmt.Sprintf("flow \"%s\" app=\"%s\" max limit %d goto [%s]", i.Name(), req.Name, req.limiter.max, req.limiter.failover))
				if !i.Goto(req.limiter.failover) {
					return
				}
				continue
			} else {
				req.limiter.AddIteration()
			}
		}

		s = time.Now()

		select {
		case <-ctx.Done():
			i.SetCancel()
			return
		case res := <-handler.Request(ctx, i, req):
			if req.log != nil {
				i.PushSteepLog(req.log.Name, s.UnixNano()/1000)
			}
			if res.Err != nil {
				wlog.Error(fmt.Sprintf("\"%s\" %v [%v] - %s (%s)", i.Name(), req.Id(), req.Args(), res.Err.Error(), time.Since(s)))
			} else {
				wlog.Debug(fmt.Sprintf("\"%s\" %v [%v] - %s (%s)", i.Name(), req.Id(), req.Args(), res.Res.String(), time.Since(s)))
			}

			if req.IsCancel() {
				i.SetCancel()
				wlog.Debug(fmt.Sprintf("flow \"%s\" set break from application \"%s\"", i.Name(), req.Name))
			}

			if i.IsCancel() {
				wlog.Debug(fmt.Sprintf("flow \"%s\" break", i.Name()))
				return
			}
		}
	}
}
