package im

import (
	"context"
	"fmt"

	"github.com/webitel/flow_manager/flow"
	"github.com/webitel/flow_manager/gen/cc"
	"github.com/webitel/flow_manager/model"
)

type Queue struct {
	Id   int32  `json:"id"`
	Name string `json:"name"`
}

type QueueJoinArg struct {
	Priority int32 `json:"priority"`
	Bucket   struct {
		Id int32 `json:"id"`
	} `json:"bucket"`
	Queue Queue `json:"queue"`
	Agent *struct {
		Id        *int32  `json:"id"`
		Extension *string `json:"extension"`
	}
	Timers []flow.TimerArgs `json:"timers"`
}

func (r *Router) cancelQueue(ctx context.Context, scope *flow.Flow, conn Dialog, args any) (model.Response, *model.AppError) {
	key := conn.GetQueueKey()
	if key == nil {
		// TODO NO QUEUE
		return model.CallResponseError, nil
	}

	err := r.fm.CancelAttempt(ctx, *key, "cancel")
	if err != nil {
		return nil, err
	}

	return conn.Set(ctx, model.Variables{
		"cc_cancel": fmt.Sprintf("%v", conn.SetQueue(nil)),
	})
}

func (r *Router) joinQueue(ctx context.Context, scope *flow.Flow, conn Dialog, args any) (model.Response, *model.AppError) {
	var q QueueJoinArg
	var wCancel context.CancelFunc
	var wCtx context.Context
	var stickyAgentId int32

	if err := r.Decode(scope, args, &q); err != nil {
		return nil, err
	}

	if q.Queue.Id == 0 && q.Queue.Name != "" {
		var err *model.AppError
		if q.Queue.Id, err = r.fm.FindQueueByName(conn.DomainId(), q.Queue.Name); err != nil {
			return nil, err
		}
	}

	wCtx, wCancel = context.WithCancel(ctx)

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

	if q.Agent != nil {
		if q.Agent.Extension != nil && q.Agent.Id == nil {
			q.Agent.Id, _ = r.fm.GetAgentIdByExtension(conn.DomainId(), *q.Agent.Extension)
		}

		if q.Agent.Id != nil {
			stickyAgentId = *q.Agent.Id
		}
	}

	from := conn.From()
	to := conn.To()
	lastMsg := conn.LastMessage()

	attId, ch, err := r.fm.JoinIMToInboundQueue(ctx, &cc.IMJoinToQueueRequest{
		ThreadId: conn.ThreadId(),
		Queue: &cc.IMJoinToQueueRequest_Queue{
			Id:   q.Queue.Id,
			Name: q.Queue.Name,
		},
		Priority: q.Priority,
		BucketId: q.Bucket.Id,
		// Variables:     conn.DumpExportVariables(),
		DomainId:      conn.DomainId(),
		StickyAgentId: stickyAgentId,
		Member: &cc.IMJoinToQueueRequest_Member{
			From: &cc.IMJoinToQueueRequest_Endpoint{
				Name: from.Name,
				Sub:  from.Sub,
			},
			To: &cc.IMJoinToQueueRequest_Endpoint{
				Name: to.Name,
				Sub:  to.Sub,
			},

			LastMsg: lastMsg.Text,
			LastSub: lastMsg.From.Sub, // TODO
		},
	})
	if err != nil {
		conn.Log().Error(err.Error())

		return model.CallResponseOK, nil
	}

	defer func() {
		conn.SetQueue(nil)
		r.fm.LeavingIMToInboundQueue(attId)
	}()

	for {
		select {
		case <-ctx.Done():
			return model.CallResponseOK, nil
		case e, _ := <-ch:
			switch e.Event {
			case "bridged":
				wCancel()
			case "leaving":
				return model.CallResponseOK, nil
			}
		}
	}
}
