// Package im provides native ops for IM-channel operations that require access
// to the active IMDialog connection and the CC (contact-centre) back-end.
package im

import (
	"context"
	"fmt"

	genpb "github.com/webitel/flow_manager/gen/cc"
	domcc "github.com/webitel/flow_manager/internal/domain/cc"
	"github.com/webitel/flow_manager/internal/runtime/ops"
	"github.com/webitel/flow_manager/internal/runtime/ops/legacy"
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

// Register adds cancelQueue and joinQueue to reg.
func Register(reg *ops.Registry, deps QueueDeps) {
	reg.Register("cancelQueue", &cancelQueueOp{deps: deps})
	reg.Register("joinQueue", &joinQueueOp{deps: deps})
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
	// Timers are decoded but not yet executed in the native runtime.
	// TODO: support timers when native sub-flow branching is available.
	Timers []map[string]any `json:"timers"`
}

func (o *joinQueueOp) Execute(ctx context.Context, in ops.OpInput) (ops.OpOutput, error) {
	var argv joinQueueArgs
	if err := ops.DecodeArgs(in, &argv); err != nil {
		return ops.OpOutput{}, err
	}

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

	defer func() {
		dialog.SetQueue(nil)
		o.deps.LeavingIMToInboundQueue(attId)
	}()

	for {
		select {
		case <-ctx.Done():
			return ops.OpOutput{}, nil
		case e := <-ch:
			switch e.Event {
			case "bridged":
				// agent answered; timers (wCancel) not supported yet — see TODO above
			case "leaving":
				if e.Result != "" {
					return ops.OpOutput{
						SetVars: map[string]string{"cc_result": e.Result},
					}, nil
				}
				return ops.OpOutput{}, nil
			}
		}
	}
}
