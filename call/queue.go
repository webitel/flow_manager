package call

import (
	"context"
	"fmt"
	"github.com/webitel/call_center/grpc_api/cc"
	"github.com/webitel/flow_manager/flow"
	"github.com/webitel/flow_manager/model"
	"github.com/webitel/wlog"
	"io"
	"time"
)

/*
   {
       "joinQueue": {
           "bucket_id": null,
           "joined": [
               {
                   "sleep": "1000"
               }
           ],
           "name": "DEFAULT FROM",
           "number": "DEFAULT FROM",
           "priority": 1,
           "queue_id": 11,
           "queue_name": "INBOUND"
       }
   },
*/

type Queue struct {
	Id   int32
	Name string
}

type WaitingMusic struct {
	Id   int32
	Name string
	Type string
}

type QueueJoinArg struct {
	Name      string        `json:"name"`
	Number    string        `json:"number"`
	Priority  int32         `json:"priority"`
	Queue     Queue         `json:"queue"`
	Ringtone  WaitingMusic  `json:"ringtone"`
	Waiting   []interface{} `json:"waiting"`
	Reporting []interface{} `json:"reporting"`
	Timers    []TimerArgs   `json:"timers"`
}

type TimerArgs struct {
	name     string
	Interval int           `json:"interval"`
	Tries    int           `json:"tries"`
	Offset   int           `json:"offset"`
	Actions  []interface{} `json:"actions"`
}

func (r *Router) queue(ctx context.Context, scope *flow.Flow, call model.Call, args interface{}) (model.Response, *model.AppError) {
	var q QueueJoinArg

	if err := r.Decode(scope, args, &q); err != nil {
		return nil, err
	}

	var wCancel context.CancelFunc
	var wCtx context.Context
	wCtx, wCancel = context.WithCancel(ctx)

	if len(q.Waiting) > 0 {
		go flow.Route(wCtx, scope.Fork("queue-waiting", flow.ArrInterfaceToArrayApplication(q.Waiting)), r)
	}

	//call.Set(ctx, model.Variables{
	//	"exec_after_bridge_app": "socket",
	//	"exec_after_bridge_arg": `10.10.10.25:10030`,
	//})

	if len(q.Timers) > 0 {
		for k, t := range q.Timers {
			t.name = fmt.Sprintf("queue-timer-%d", k)
			go r.timer(wCtx, t, scope)
		}
	}

	defer func() {
		if wCancel != nil {
			wCancel()
			wCancel = nil
		}
	}()

	ctx2 := context.Background()
	res, err := r.fm.JoinToInboundQueue(ctx2, &cc.CallJoinToQueueRequest{
		MemberCallId: call.Id(),
		Queue: &cc.CallJoinToQueueRequest_Queue{
			Id:   q.Queue.Id,
			Name: q.Queue.Name,
		},
		WaitingMusic: &cc.CallJoinToQueueRequest_WaitingMusic{
			Id:   q.Ringtone.Id,
			Name: q.Ringtone.Name,
			Type: q.Ringtone.Type,
		},
		Priority:  q.Priority,
		Variables: call.DumpExportVariables(),
		DomainId:  call.DomainId(),
	})

	if err != nil {
		wlog.Error(err.Error())
		return model.CallResponseOK, nil
	}

	for {
		var msg cc.QueueEvent
		err = res.RecvMsg(&msg)
		if err == io.EOF {
			break
		} else if err != nil {
			wlog.Error(err.Error())
			return model.CallResponseError, nil
		}

		switch msg.Data.(type) {
		case *cc.QueueEvent_Bridged:
			if wCancel != nil {
				wCancel()
				wCancel = nil
			}

		case *cc.QueueEvent_Leaving:
			if len(q.Reporting) > 0 {
				call.Set(ctx, model.Variables{
					"cc_result": msg.Data.(*cc.QueueEvent_Leaving).Leaving.Result,
				})
				flow.Route(context.Background(), scope.Fork("queue-reporting", flow.ArrInterfaceToArrayApplication(q.Reporting)), r)
			}
			break
		}
	}

	return model.CallResponseOK, nil
}

func (r *Router) timer(ctx context.Context, t TimerArgs, scope *flow.Flow) {
	if t.Interval == 0 {
		// TODO set default ?
		return
	}

	if t.Tries == 0 {
		// todo set default ?
		t.Tries = 999
	}

	interval := time.Duration(t.Interval)
	timer := time.NewTimer(time.Second * interval)
	tries := 0
	defer wlog.Debug(fmt.Sprintf("timer [%s] stopped", t.name))
	f := scope.Fork(t.name, flow.ArrInterfaceToArrayApplication(t.Actions))

	for {
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
			tries++
			flow.Route(ctx, f, r)

			interval += time.Duration(t.Offset)
			if tries >= t.Tries || interval < 1 {
				timer.Stop()
				return
			}
			timer = time.NewTimer(time.Second * interval)
		}
	}
}
