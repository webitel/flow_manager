package chat

import (
	"context"
	"fmt"
	"io"
	"strconv"
	"time"

	genpb "github.com/webitel/flow_manager/gen/cc"
	"github.com/webitel/flow_manager/internal/runtime/ops"
	"github.com/webitel/flow_manager/internal/runtime/tree"
	"github.com/webitel/flow_manager/model"
)

// QueueDeps is the subset of  the queue ops need.
type QueueDeps interface {
	CancelAttempt(ctx context.Context, att model.InQueueKey, result string) *model.AppError
	FindQueueByName(domainId int64, name string) (int32, *model.AppError)
	GetAgentIdByExtension(domainId int64, extension string) (*int32, *model.AppError)
	JoinChatToInboundQueue(ctx context.Context, in *genpb.ChatJoinToQueueRequest) (genpb.MemberService_ChatJoinToQueueClient, error)
}

// RegisterQueue adds cancelQueue and joinQueue to reg.
func RegisterQueue(reg *ops.Registry, deps QueueDeps) {
	reg.Register("cancelQueue", &cancelQueueOp{deps: deps})
	reg.Register("joinQueue", &joinQueueOp{deps: deps})
}

// ── cancelQueue ───────────────────────────────────────────────────────────────

type cancelQueueOp struct{ deps QueueDeps }

func (o *cancelQueueOp) Kind() ops.OpKind { return ops.OpKindSync }

func (o *cancelQueueOp) Execute(ctx context.Context, in ops.OpInput) (ops.OpOutput, error) {
	conv, ok := conversationFromContext(ctx)
	if !ok {
		return ops.OpOutput{}, fmt.Errorf("cancelQueue: no conversation in context")
	}
	key := conv.GetQueueKey()
	if key == nil {
		return ops.OpOutput{}, fmt.Errorf("cancelQueue: no active queue")
	}
	if appErr := o.deps.CancelAttempt(ctx, *key, "cancel"); appErr != nil {
		return ops.OpOutput{}, fmt.Errorf("cancelQueue: %s", appErr.Error())
	}
	wasSet := conv.SetQueue(nil)
	return ops.OpOutput{
		SetVars: map[string]string{
			"cc_cancel": strconv.FormatBool(wasSet),
		},
	}, nil
}

// ── joinQueue — blocking sync, no suspend/resume ──────────────────────────────
// Mirrors legacy behavior: blocks until the gRPC stream closes (EOF or error).
// When the chat CC service gains proper event channels (like IM), this op can
// be replaced with an OpKindSuspendable version without changing the schema.

type joinQueueOp struct{ deps QueueDeps }

func (o *joinQueueOp) Kind() ops.OpKind { return ops.OpKindSync }

type joinQueueArgs struct {
	Priority int32 `json:"priority"`
	Bucket   struct {
		Id int32 `json:"id"`
	} `json:"bucket"`
	Queue struct {
		Id   int32  `json:"id"`
		Name string `json:"name"`
	} `json:"queue"`
	Agent *struct {
		Id        *int32  `json:"id"`
		Extension *string `json:"extension"`
	} `json:"agent"`
	Timers []chatTimerArg `json:"timers"`
}

type chatTimerArg struct {
	Interval    int `json:"interval"`
	Tries       int `json:"tries"`
	Offset      int `json:"offset"`
	ChildrenIdx int `json:"_children_idx"`
}

func (o *joinQueueOp) Execute(ctx context.Context, in ops.OpInput) (ops.OpOutput, error) {
	var argv joinQueueArgs
	if err := ops.DecodeArgs(in, &argv); err != nil {
		return ops.OpOutput{}, err
	}

	conv, ok := conversationFromContext(ctx)
	if !ok {
		return ops.OpOutput{}, fmt.Errorf("joinQueue: no conversation in context")
	}

	if argv.Queue.Id == 0 && argv.Queue.Name != "" {
		id, err := o.deps.FindQueueByName(in.DomainID, argv.Queue.Name)
		if err != nil {
			return ops.OpOutput{}, fmt.Errorf("joinQueue: find queue: %s", err.Error())
		}
		argv.Queue.Id = id
	}

	var stickyAgentId int32
	if argv.Agent != nil {
		if argv.Agent.Extension != nil && argv.Agent.Id == nil {
			argv.Agent.Id, _ = o.deps.GetAgentIdByExtension(in.DomainID, *argv.Agent.Extension)
		}
		if argv.Agent.Id != nil {
			stickyAgentId = *argv.Agent.Id
		}
	}

	// timerCtx is cancelled on bridged so timers stop when agent connects.
	timerCtx, timerCancel := context.WithCancel(ctx)
	defer timerCancel()

	stream, err := o.deps.JoinChatToInboundQueue(ctx, &genpb.ChatJoinToQueueRequest{
		ConversationId: conv.Id(),
		Queue: &genpb.ChatJoinToQueueRequest_Queue{
			Id:   argv.Queue.Id,
			Name: argv.Queue.Name,
		},
		Priority:      argv.Priority,
		BucketId:      argv.Bucket.Id,
		Variables:     conv.DumpExportVariables(),
		DomainId:      in.DomainID,
		StickyAgentId: stickyAgentId,
	})
	if err != nil {
		return ops.OpOutput{}, nil
	}

	startChatTimers(timerCtx, argv.Timers, in)
	defer conv.SetQueue(nil)

	setVars := make(map[string]string)

	for {
		var msg genpb.QueueEvent
		if recvErr := stream.RecvMsg(&msg); recvErr != nil {
			if recvErr != io.EOF {
				return ops.OpOutput{SetVars: setVars}, nil
			}
			break
		}

		switch e := msg.Data.(type) {
		case *genpb.QueueEvent_Joined:
			conv.SetQueue(&model.InQueueKey{
				AttemptId: e.Joined.GetAttemptId(),
				AppId:     e.Joined.GetAppId(),
			})

		case *genpb.QueueEvent_Offering:
			name := e.Offering.GetAgentName()
			id := strconv.Itoa(int(e.Offering.GetAgentId()))
			_, _ = conv.Set(ctx, model.Variables{"cc_agent_name": name, "cc_agent_id": id})
			setVars["cc_agent_name"] = name
			setVars["cc_agent_id"] = id

		case *genpb.QueueEvent_Missed:
			_, _ = conv.Set(ctx, model.Variables{"cc_agent_name": "", "cc_agent_id": ""})
			setVars["cc_agent_name"] = ""
			setVars["cc_agent_id"] = ""

		case *genpb.QueueEvent_Bridged:
			timerCancel()

		case *genpb.QueueEvent_Leaving:
			result := e.Leaving.GetResult()
			_, _ = conv.Set(ctx, model.Variables{"cc_result": result})
			setVars["cc_result"] = result
			return ops.OpOutput{SetVars: setVars}, nil
		}
	}

	return ops.OpOutput{SetVars: setVars}, nil
}

func startChatTimers(ctx context.Context, timers []chatTimerArg, in ops.OpInput) {
	if in.RunBranch == nil || in.Node == nil || len(timers) == 0 {
		return
	}
	varSnap := make(map[string]string, len(in.Variables))
	for k, v := range in.Variables {
		varSnap[k] = v
	}
	for _, t := range timers {
		t := t
		if t.Interval <= 0 || t.ChildrenIdx < 0 || t.ChildrenIdx >= len(in.Node.Children) {
			continue
		}
		branch := in.Node.Children[t.ChildrenIdx]
		go runChatTimer(ctx, t, branch, varSnap, in.RunBranch)
	}
}

func runChatTimer(ctx context.Context, t chatTimerArg, branch *tree.Node, varSnap map[string]string, runBranch func(context.Context, *tree.Node, map[string]string)) {
	tries := t.Tries
	if tries <= 0 {
		tries = 999
	}
	interval := time.Duration(t.Interval) * time.Second
	timer := time.NewTimer(interval)
	defer timer.Stop()
	for fired := 0; fired < tries; fired++ {
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
			runBranch(ctx, branch, varSnap)
			interval += time.Duration(t.Offset) * time.Second
			if interval < time.Second {
				return
			}
			timer.Reset(interval)
		}
	}
}
