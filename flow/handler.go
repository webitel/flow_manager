package flow

import (
	"context"
	"fmt"
	"time"

	"github.com/webitel/flow_manager/model"
	"github.com/webitel/wlog"
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

	i.log.Debug("start flow " + i.name)
	defer i.log.Debug("stop flow " + i.name)

	for {
		req = i.NextRequest()
		if req == nil {
			return
		}

		if i.IsCancel() {
			i.log.Debug("break")
			return
		}

		if req.limiter != nil {
			if req.limiter.MaxCount() {
				i.log.Debug(fmt.Sprintf("max limit %d goto [%s]", req.limiter.max, req.limiter.failover))
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
				i.log.Warn("application "+req.Id()+", error: "+res.Err.Error(),
					wlog.String("method", req.Id()),
					wlog.Any("args", req.Args()),
					wlog.Duration("duration", time.Since(s)),
				)
			} else {
				i.log.Debug("application "+req.Id()+", success: "+res.Res.String(),
					wlog.String("method", req.Id()),
					wlog.Any("args", req.Args()),
					wlog.Duration("duration", time.Since(s)),
				)
			}

			if req.IsCancel() {
				i.SetCancel()
				i.log.Debug("set break from application"+req.Name,
					wlog.String("method", req.Name),
				)
			}

			if i.IsCancel() {
				i.log.Debug("set break from flow")
				return
			}
		}
	}
}
