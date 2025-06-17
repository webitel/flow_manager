package call

import (
	"context"
	"fmt"
	"io"

	"github.com/webitel/flow_manager/flow"
	"github.com/webitel/flow_manager/gen/cc"
	"github.com/webitel/flow_manager/model"
	"github.com/webitel/wlog"
)

type Queue struct {
	Id   int    `json:"id"`
	Name string `json:"name"`
}

type WaitingMusic struct {
	Id   *int32
	Name *string
	Type *string
}

type QueueJoinArg struct {
	Name     string `json:"name"`
	Number   string `json:"number"`
	Priority int32  `json:"priority"`
	Queue    Queue  `json:"queue"`
	BucketId int32  `json:"bucket_id"` // deprecated
	Bucket   struct {
		Id int32 `json:"id"`
	} `json:"bucket"`
	Agent *struct {
		Id        *int32  `json:"id"`
		Extension *string `json:"extension"`
	}
	StickyAgentId       int32               `json:"stickyAgentId"`
	Ringtone            model.PlaybackFile  `json:"ringtone"`
	Waiting             []interface{}       `json:"waiting"`
	Reporting           []interface{}       `json:"reporting"`
	Offering            []interface{}       `json:"offering"`
	Bridged             []interface{}       `json:"bridged"`
	Timers              []flow.TimerArgs    `json:"timers"`
	TransferAfterBridge *model.SearchEntity `json:"transferAfterBridge"`
}

func (r *Router) cancelQueue(ctx context.Context, scope *flow.Flow, call model.Call, args interface{}) (model.Response, *model.AppError) {
	return call.Set(ctx, model.Variables{
		"cc_cancel": fmt.Sprintf("%v", call.CancelQueue()),
	})
}

func (r *Router) queue(ctx context.Context, scope *flow.Flow, call model.Call, args interface{}) (model.Response, *model.AppError) {
	var q QueueJoinArg
	var stickyAgentId int32

	if call.InQueue() {
		return nil, model.NewInternalError("call.queue.in_queue", "call is in queue")
	}

	if err := r.Decode(scope, args, &q); err != nil {
		return nil, err
	}

	var wCancel context.CancelFunc
	var wCtx context.Context
	wCtx, wCancel = context.WithCancel(ctx)

	if len(q.Waiting) > 0 {
		go flow.Route(wCtx, scope.Fork("queue-waiting", flow.ArrInterfaceToArrayApplication(q.Waiting)), r)
	}

	if q.BucketId > 0 && q.Bucket.Id == 0 {
		q.Bucket.Id = q.BucketId
	}

	if len(q.Timers) > 0 {
		for k, t := range q.Timers {
			t.Name = fmt.Sprintf("queue-timer-%d", k)
			go scope.Timer(wCtx, t, r)
		}
	}

	defer func() {
		if wCancel != nil {
			wCancel()
			wCancel = nil
		}
	}()

	if q.TransferAfterBridge != nil && q.TransferAfterBridge.Id != nil {
		if _, err := call.SetTransferAfterBridge(ctx, *q.TransferAfterBridge.Id); err != nil {
			return nil, err
		}
	}

	t := call.GetVariable("variable_transfer_history")
	var ringtone *cc.CallJoinToQueueRequest_WaitingMusic

	//FIXME
	if q.Ringtone.Name != nil || q.Ringtone.Id != nil {
		var err *model.AppError
		req := make([]*model.PlaybackFile, 1, 1)
		req[0] = &model.PlaybackFile{
			Id:   q.Ringtone.Id,
			Name: q.Ringtone.Name,
		}
		if req, err = r.fm.GetMediaFiles(call.DomainId(), &req); err != nil {
			return nil, err
		} else if req != nil && req[0] != nil && req[0].Type != nil {
			ringtone = &cc.CallJoinToQueueRequest_WaitingMusic{
				Id:   int32(*req[0].Id),
				Type: *req[0].Type,
			}
		}
	}

	if q.Agent != nil {
		if q.Agent.Extension != nil && q.Agent.Id == nil {
			q.Agent.Id, _ = r.fm.GetAgentIdByExtension(call.DomainId(), *q.Agent.Extension)
		}

		if q.Agent.Id != nil {
			stickyAgentId = *q.Agent.Id
		}
	} else {
		stickyAgentId = q.StickyAgentId
	}
	vars := call.DumpExportVariables()

	if cid := call.GetContactId(); cid != 0 {
		vars["wbt_contact_id"] = fmt.Sprintf("%d", cid)
	}

	/*
		l := scope.Logs()
		if len(l) > 0 {
			//var res []*model.StepLog
			//scope.Decode(scope.Logs(), res)1
			d, _ := json.Marshal(l)
			vars["wbt_ivr_log"] = string(d)
		}
	*/

	// TODO
	if call.Stopped() {
		return model.CallResponseError, nil
	}

	if call.HangupCause() != "" {
		return nil, model.NewAppError("Call", "call.queue.join.hangup", nil, "Call is down", 500)
	}

	ctx2, cancelQueue := context.WithCancel(context.Background())
	res, err := r.fm.JoinToInboundQueue(ctx2, &cc.CallJoinToQueueRequest{
		MemberCallId: call.Id(),
		Queue: &cc.CallJoinToQueueRequest_Queue{
			Id:   int32(q.Queue.Id),
			Name: q.Queue.Name,
		},
		WaitingMusic:  ringtone,
		Priority:      q.Priority,
		BucketId:      q.Bucket.Id,
		Variables:     vars,
		DomainId:      call.DomainId(),
		StickyAgentId: stickyAgentId,
		IsTransfer:    call.TransferQueueId() > 0 && !call.IsBlindTransferQueue(),
	})

	if err != nil {
		call.Log().Err(err)
		return model.CallResponseOK, nil
	}

	call.SetQueueCancel(cancelQueue)
	defer call.SetQueueCancel(nil)

	// TODO bug close stream channel
	for {
		var msg cc.QueueEvent
		err = res.RecvMsg(&msg)
		if err == io.EOF {
			break
		} else if err != nil {
			wlog.Error(err.Error())
			return model.CallResponseError, nil
		}

		switch e := msg.Data.(type) {
		case *cc.QueueEvent_Offering:
			if len(q.Offering) > 0 {
				call.Set(ctx, model.Variables{
					"cc_agent_name":    e.Offering.AgentName,
					"cc_agent_call_id": e.Offering.AgentCallId,
					"cc_agent_id":      fmt.Sprintf("%d", e.Offering.AgentId),
				})
				flow.Route(context.Background(), scope.Fork("queue-offering", flow.ArrInterfaceToArrayApplication(q.Offering)), r)
			}
		case *cc.QueueEvent_Bridged:
			call.SetQueueCancel(nil)
			if wCancel != nil {
				wCancel()
				wCancel = nil
				if len(q.Bridged) > 0 {
					flow.Route(context.Background(), scope.Fork("queue-bridged", flow.ArrInterfaceToArrayApplication(q.Bridged)), r)
				}
			}

		case *cc.QueueEvent_Leaving:
			call.Set(ctx, model.Variables{
				"cc_result": e.Leaving.Result,
			})
			if len(q.Reporting) > 0 {
				flow.Route(context.Background(), scope.Fork("queue-reporting", flow.ArrInterfaceToArrayApplication(q.Reporting)), r)
			}
			break
		}
	}

	if t != call.GetVariable("variable_transfer_history") {
		scope.SetCancel()
	}

	return model.CallResponseOK, nil
}
