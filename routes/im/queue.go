package im

import (
	"context"
	"fmt"
	"io"

	"github.com/webitel/flow_manager/flow"
	"github.com/webitel/flow_manager/gen/cc"
	"github.com/webitel/flow_manager/model"
)

type Queue struct {
	Id   int32  `json:"id"`
	Name string `json:"name"`
}

type Agent struct {
	ID        *int32  `json:"id"`
	Extension *string `json:"extension"`
}

type Bucket struct {
	ID int32 `json:"id"`
}

type QueueJoinArg struct {
	Priority int32            `json:"priority"`
	Bucket   Bucket           `json:"bucket"`
	Queue    Queue            `json:"queue"`
	Agent    *Agent           `json:"agent"`
	Timers   []flow.TimerArgs `json:"timers"`
	Offering []any            `json:"offering"`
	Missed   []any            `json:"missed"`
	Bridged  []any            `json:"bridged"`
}

func (r *Router) cancelQueue(ctx context.Context, scope *flow.Flow, conv Dialog, args any) (model.Response, *model.AppError) {
	key := conv.GetQueueKey()
	if key == nil {
		return model.CallResponseError, nil
	}

	err := r.fm.CancelAttempt(ctx, *key, "cancel")
	if err != nil {
		return nil, err
	}

	return conv.Set(ctx, model.Variables{
		"cc_cancel": fmt.Sprintf("%v", conv.SetQueue(nil)),
	})
}

func (r *Router) joinQueue(ctx context.Context, scope *flow.Flow, conv Dialog, args any) (model.Response, *model.AppError) {
	var q QueueJoinArg
	var wCancel context.CancelFunc
	var wCtx context.Context
	var stickyAgentId int32

	wCtx, wCancel = context.WithCancel(ctx)

	if err := r.Decode(scope, args, &q); err != nil {
		wCancel()
		return nil, err
	}

	if q.Queue.Id == 0 && q.Queue.Name != "" {
		var err *model.AppError
		if q.Queue.Id, err = r.fm.FindQueueByName(conv.DomainId(), q.Queue.Name); err != nil {
			wCancel()
			return nil, err
		}
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

	if q.Agent != nil {
		if q.Agent.Extension != nil && q.Agent.ID == nil {
			q.Agent.ID, _ = r.fm.GetAgentIdByExtension(conv.DomainId(), *q.Agent.Extension)
		}

		if q.Agent.ID != nil {
			stickyAgentId = *q.Agent.ID
		}
	}

	res, err := r.fm.JoinChatToInboundQueue(ctx, &cc.ChatJoinToQueueRequest{
		ConversationId: conv.Id(),
		Queue: &cc.ChatJoinToQueueRequest_Queue{
			Id:   q.Queue.Id,
			Name: q.Queue.Name,
		},
		Priority:      q.Priority,
		BucketId:      q.Bucket.ID,
		Variables:     conv.DumpExportVariables(),
		DomainId:      conv.DomainId(),
		StickyAgentId: stickyAgentId,
	})
	if err != nil {
		conv.Log().Error(err.Error())
		return model.CallResponseOK, nil
	}

	defer conv.SetQueue(nil)

	for {
		var msg cc.QueueEvent
		e := res.RecvMsg(&msg)
		if e == io.EOF {
			break
		} else if e != nil {
			conv.Log().Error(e.Error())
			return model.CallResponseError, nil
		}

		switch ev := msg.Data.(type) {
		case *cc.QueueEvent_Joined:
			conv.SetQueue(&model.InQueueKey{
				AttemptId: ev.Joined.AttemptId,
				AppId:     ev.Joined.AppId,
			})
		case *cc.QueueEvent_Offering:
			conv.Set(ctx, model.Variables{
				"cc_agent_name": ev.Offering.AgentName,
				"cc_agent_id":   ev.Offering.AgentId,
			})
			if len(q.Offering) > 0 {
				go flow.Route(wCtx, scope.Fork("queue-offering", flow.ArrInterfaceToArrayApplication(q.Offering)), r)
			}
		case *cc.QueueEvent_Missed:
			conv.Set(ctx, model.Variables{
				"cc_agent_name": "",
				"cc_agent_id":   "",
			})
			if len(q.Missed) > 0 {
				go flow.Route(wCtx, scope.Fork("queue-missed", flow.ArrInterfaceToArrayApplication(q.Missed)), r)
			}
		case *cc.QueueEvent_Bridged:
			conv.SetQueue(nil)
			if len(q.Bridged) > 0 {
				flow.Route(wCtx, scope.Fork("queue-bridged", flow.ArrInterfaceToArrayApplication(q.Bridged)), r)
			}
			if wCancel != nil {
				wCancel()
				wCancel = nil
			}
		case *cc.QueueEvent_Leaving:
			conv.Set(ctx, model.Variables{
				"cc_result": ev.Leaving.Result,
			})
		}
	}

	return model.CallResponseOK, nil
}
