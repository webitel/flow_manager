package call

import (
	"context"
	"fmt"
	"github.com/webitel/flow_manager/flow"
	"github.com/webitel/flow_manager/model"
	"github.com/webitel/protos/cc"
	"github.com/webitel/wlog"
	"io"
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
	Name                string              `json:"name"`
	Number              string              `json:"number"`
	Priority            int32               `json:"priority"`
	Queue               Queue               `json:"queue"`
	BucketId            int32               `json:"bucket_id"` // TODO
	StickyAgentId       int32               `json:"stickyAgentId"`
	Ringtone            model.PlaybackFile  `json:"ringtone"`
	Waiting             []interface{}       `json:"waiting"`
	Reporting           []interface{}       `json:"reporting"`
	Timers              []flow.TimerArgs    `json:"timers"`
	TransferAfterBridge *model.SearchEntity `json:"transferAfterBridge"`
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

	ctx2 := context.Background()
	res, err := r.fm.JoinToInboundQueue(ctx2, &cc.CallJoinToQueueRequest{
		MemberCallId: call.Id(),
		Queue: &cc.CallJoinToQueueRequest_Queue{
			Id:   int32(q.Queue.Id),
			Name: q.Queue.Name,
		},
		WaitingMusic:  ringtone,
		Priority:      q.Priority,
		BucketId:      q.BucketId,
		Variables:     call.DumpExportVariables(),
		DomainId:      call.DomainId(),
		StickyAgentId: q.StickyAgentId,
	})

	if err != nil {
		wlog.Error(err.Error())
		return model.CallResponseOK, nil
	}

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

		switch msg.Data.(type) {
		case *cc.QueueEvent_Bridged:
			if wCancel != nil {
				wCancel()
				wCancel = nil
			}

		case *cc.QueueEvent_Leaving:
			call.Set(ctx, model.Variables{
				"cc_result": msg.Data.(*cc.QueueEvent_Leaving).Leaving.Result,
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
