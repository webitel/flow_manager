// Package im provides native ops for IM-channel operations that require access
// to the active IMDialog connection and the CC (contact-centre) back-end.
package im

import (
	"context"
	"fmt"
	"strconv"
	"time"

	genpb "github.com/webitel/flow_manager/gen/cc"
	domcc "github.com/webitel/flow_manager/internal/domain/cc"
	"github.com/webitel/flow_manager/internal/runtime/ops"
	"github.com/webitel/flow_manager/internal/runtime/ops/legacy"
	"github.com/webitel/flow_manager/internal/runtime/state"
	"github.com/webitel/flow_manager/internal/runtime/tree"
	"github.com/webitel/flow_manager/model"
)

// QueueDeps is the subset of ports.RouterDeps that the queue ops need.
type QueueDeps interface {
	CancelAttempt(ctx context.Context, att model.InQueueKey, result string) *model.AppError
	FindQueueByName(domainId int64, name string) (int32, *model.AppError)
	GetAgentIdByExtension(domainId int64, extension string) (*int32, *model.AppError)
	JoinIMToInboundQueue(ctx context.Context, in *genpb.IMJoinToQueueRequest) (int64, <-chan domcc.QueueEvent, error)
	LeavingIMToInboundQueue(attId int64)
}

// QueueDispatcher dispatches a resume event to a suspended flow.
// coordinator.Coordinator satisfies this interface.
type QueueDispatcher interface {
	Dispatch(ctx context.Context, resumeKey string, payload map[string]string) error
}

// DispatchFunc is a function adapter for QueueDispatcher.
// It allows passing a closure that captures a coordinator variable that may be
// set after Register is called (late-binding pattern for Bootstrap wiring).
type DispatchFunc func(ctx context.Context, resumeKey string, payload map[string]string) error

func (f DispatchFunc) Dispatch(ctx context.Context, resumeKey string, payload map[string]string) error {
	return f(ctx, resumeKey, payload)
}

// Register adds cancelQueue and joinQueue to reg.
// coord is used by joinQueue to dispatch CC events to the suspended flow.
func Register(reg *ops.Registry, deps QueueDeps, coord QueueDispatcher) {
	reg.Register("cancelQueue", &cancelQueueOp{deps: deps})
	reg.Register("joinQueue", &joinQueueOp{deps: deps, coord: coord})
}

// dialogFromContext retrieves the IMDialog stored by legacy.WithConnection.
// Returns nil when no connection or the connection is not an IMDialog.
func dialogFromContext(ctx context.Context) (model.IMDialog, bool) {
	conn := legacy.ConnectionFromContext(ctx)
	if conn == nil {
		return nil, false
	}
	d, ok := conn.(model.IMDialog)
	return d, ok
}

// ── cancelQueue ───────────────────────────────────────────────────────────────

type cancelQueueOp struct{ deps QueueDeps }

func (o *cancelQueueOp) Kind() ops.OpKind { return ops.OpKindSync }

func (o *cancelQueueOp) Execute(ctx context.Context, in ops.OpInput) (ops.OpOutput, error) {
	dialog, ok := dialogFromContext(ctx)
	if !ok {
		return ops.OpOutput{}, fmt.Errorf("cancelQueue: no IMDialog in context")
	}

	key := dialog.GetQueueKey()
	if key == nil {
		return ops.OpOutput{}, fmt.Errorf("cancelQueue: no active queue")
	}

	if err := o.deps.CancelAttempt(ctx, *key, "cancel"); err != nil {
		return ops.OpOutput{}, fmt.Errorf("cancelQueue: %s", err.Error())
	}

	wasSet := dialog.SetQueue(nil)
	return ops.OpOutput{
		SetVars: map[string]string{
			"cc_cancel": fmt.Sprintf("%v", wasSet),
		},
	}, nil
}

// ── joinQueue ─────────────────────────────────────────────────────────────────

type joinQueueOp struct {
	deps  QueueDeps
	coord QueueDispatcher
}

func (o *joinQueueOp) Kind() ops.OpKind { return ops.OpKindSuspendable }

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
	// Timers are populated by the tree parser: "actions" is extracted into a
	// child container node and replaced with "_children_idx" so Execute can
	// look up in.Node.Children[ChildrenIdx].
	Timers []timerArg `json:"timers"`
}

// timerArg mirrors flow.TimerArgs for the native runtime.
// Interval is the initial delay in seconds. After each fire the next delay is
// Interval += Offset (growing intervals). Tries caps the repetitions (0 = 999).
type timerArg struct {
	Interval    int `json:"interval"`
	Tries       int `json:"tries"`
	Offset      int `json:"offset"`
	ChildrenIdx int `json:"_children_idx"`
}

// ccEventKey and ccResultKey are the payload field names used by the CC event
// goroutine to communicate queue state changes to the resume path.
const (
	ccEventKey  = "cc_event"
	ccResultKey = "cc_result"
)

func (o *joinQueueOp) Execute(ctx context.Context, in ops.OpInput) (ops.OpOutput, error) {
	var argv joinQueueArgs
	if err := ops.DecodeArgs(in, &argv); err != nil {
		return ops.OpOutput{}, err
	}

	suspendKey := "msg:" + in.ConnID

	// ── Resume path ──────────────────────────────────────────────────────────
	// The op is called again (ReenterOnResume=true) with the payload that woke
	// the flow. The payload may carry a CC event or a plain inbound message.
	if in.ResumePayload != nil {
		attIDStr := in.Variables["cc_attempt_id"]

		ccEvent := in.ResumePayload[ccEventKey]
		switch ccEvent {
		case "leaving":
			// Subscriber left the queue — only now do we exit the op.
			out := ops.OpOutput{}
			if result := in.ResumePayload[ccResultKey]; result != "" {
				out.SetVars = map[string]string{"cc_result": result}
			}
			return out, nil

		case "":
			// No CC event — plain inbound message. Check trigger commands first.
			if len(in.Triggers) > 0 {
				msg := in.ResumePayload["msg"]
				cmdKey := "commands-" + msg
				if trig, ok := in.Triggers[cmdKey]; ok {
					// Run the trigger branch inline; re-enter this op after it
					// finishes so we keep waiting for the queue result.
					return ops.OpOutput{
						Branch:          trig,
						ReenterOnResume: true,
					}, nil
				}
			}
			// Unhandled message while waiting — re-suspend on the same key.
			return ops.OpOutput{
				SuspendKey:      suspendKey,
				Pending:         buildQueuePending(suspendKey, in.Node.ID, attIDStr),
				ReenterOnResume: true,
				ReSuspend:       true,
			}, nil

		default:
			// Unknown CC event — re-suspend and wait for the next one.
			return ops.OpOutput{
				SuspendKey:      suspendKey,
				Pending:         buildQueuePending(suspendKey, in.Node.ID, attIDStr),
				ReenterOnResume: true,
				ReSuspend:       true,
			}, nil
		}
	}

	// ── Fresh path ────────────────────────────────────────────────────────────
	dialog, ok := dialogFromContext(ctx)
	if !ok {
		return ops.OpOutput{}, fmt.Errorf("joinQueue: no IMDialog in context")
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

	from := dialog.From()
	to := dialog.To()
	lastMsg := dialog.LastMessage()

	attId, ch, err := o.deps.JoinIMToInboundQueue(ctx, &genpb.IMJoinToQueueRequest{
		ThreadId: dialog.ThreadId(),
		Queue: &genpb.IMJoinToQueueRequest_Queue{
			Id:   argv.Queue.Id,
			Name: argv.Queue.Name,
		},
		Priority:      argv.Priority,
		BucketId:      argv.Bucket.Id,
		DomainId:      in.DomainID,
		StickyAgentId: stickyAgentId,
		Member: &genpb.IMJoinToQueueRequest_Member{
			From: &genpb.IMJoinToQueueRequest_Endpoint{
				Name: from.Name,
				Sub:  from.Sub,
			},
			To: &genpb.IMJoinToQueueRequest_Endpoint{
				Name: to.Name,
				Sub:  to.Sub,
			},
			LastMsg: lastMsg.Text,
			LastSub: lastMsg.From.Sub,
		},
	})
	if err != nil {
		return ops.OpOutput{}, nil
	}

	attIDStr := strconv.FormatInt(attId, 10)

	// timerCtx lives beyond this Execute call; it is cancelled by the CC
	// goroutine when the queue terminates, stopping all timer sub-flows.
	timerCtx, timerCancel := context.WithCancel(ctx)
	startTimers(timerCtx, argv.Timers, in)

	coord := o.coord
	deps := o.deps
	go func() {
		defer timerCancel()
		defer func() {
			dialog.SetQueue(nil)
			deps.LeavingIMToInboundQueue(attId)
		}()
		for e := range ch {
			if e.Event == "bridged" {
				timerCancel()
			}
			payload := map[string]string{ccEventKey: e.Event}
			if e.Result != "" {
				payload[ccResultKey] = e.Result
			}
			_ = coord.Dispatch(ctx, suspendKey, payload)
			if e.Event == "leaving" {
				return
			}
		}
	}()

	return ops.OpOutput{
		SuspendKey:      suspendKey,
		Pending:         buildQueuePending(suspendKey, in.Node.ID, attIDStr),
		ReenterOnResume: true,
		SetVars:         map[string]string{"cc_attempt_id": attIDStr},
	}, nil
}

func buildQueuePending(suspendKey, nodeID, attIDStr string) *state.PendingIntent {
	return &state.PendingIntent{
		OpName:    "joinQueue",
		NodeID:    nodeID,
		ResumeKey: suspendKey,
		Args:      map[string]string{"att_id": attIDStr},
	}
}

// startTimers launches a goroutine for each timer entry that has a parsed child
// node. Each goroutine fires every Interval seconds (growing by Offset) up to
// Tries times, executing the corresponding sub-flow via in.RunBranch.
func startTimers(ctx context.Context, timers []timerArg, in ops.OpInput) {
	if in.RunBranch == nil || in.Node == nil || len(timers) == 0 {
		return
	}
	// Snapshot variables once; the blocking Execute loop does not mutate them.
	varSnap := make(map[string]string, len(in.Variables))
	for k, v := range in.Variables {
		varSnap[k] = v
	}
	for _, t := range timers {
		t := t
		if t.Interval <= 0 {
			continue
		}
		if t.ChildrenIdx < 0 || t.ChildrenIdx >= len(in.Node.Children) {
			continue
		}
		branch := in.Node.Children[t.ChildrenIdx]
		go runTimer(ctx, t, branch, varSnap, in.RunBranch)
	}
}

func runTimer(ctx context.Context, t timerArg, branch *tree.Node, varSnap map[string]string, runBranch func(context.Context, *tree.Node, map[string]string)) {
	tries := t.Tries
	if tries <= 0 {
		tries = 999
	}
	interval := time.Duration(t.Interval) * time.Second
	timer := time.NewTimer(interval)
	defer timer.Stop()
	fired := 0
	for {
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
			runBranch(ctx, branch, varSnap)
			fired++
			if fired >= tries {
				return
			}
			interval += time.Duration(t.Offset) * time.Second
			if interval < time.Second {
				return
			}
			timer.Reset(interval)
		}
	}
}
