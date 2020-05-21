package call

import (
	"context"
	"github.com/webitel/call_center/grpc_api/cc"
	"github.com/webitel/flow_manager/flow"
	"github.com/webitel/flow_manager/model"
	"github.com/webitel/wlog"
	"io"
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
}

func (r *Router) queue(ctx context.Context, scope *flow.Flow, call model.Call, args interface{}) (model.Response, *model.AppError) {
	var q QueueJoinArg

	if err := r.Decode(scope, args, &q); err != nil {
		return nil, err
	}

	var wCancel context.CancelFunc

	if len(q.Waiting) > 0 {
		var wCtx context.Context
		wCtx, wCancel = context.WithCancel(ctx)
		go flow.Route(wCtx, scope.Fork("queue-waiting", flow.ArrInterfaceToArrayApplication(q.Waiting)), r)
	}

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
			}

		case *cc.QueueEvent_Leaving:
			if len(q.Reporting) > 0 {
				call.Set(ctx, model.Variables{
					"cc_result": msg.Data.(*cc.QueueEvent_Leaving).Leaving.Result,
				})
				flow.Route(context.Background(), scope.Fork("queue-reporting", flow.ArrInterfaceToArrayApplication(q.Reporting)), r)
			}
		}
	}

	return model.CallResponseOK, nil
}
