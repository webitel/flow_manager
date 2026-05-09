package chat

import (
	"context"
	"fmt"

	"github.com/webitel/flow_manager/internal/runtime/ops"
	"github.com/webitel/flow_manager/internal/runtime/ops/domain/messaging"
	"github.com/webitel/flow_manager/internal/runtime/state"
)

// ChatWaitable is the subset of *providers/grpc.conversation that the
// native chat recvMessage op needs. It signals the external chat server
// to forward the next inbound message.
type ChatWaitable interface {
	StartWaiting(timeout int)
}

type chatWaitableKey struct{}

// WithChatWaitable stores a ChatWaitable in ctx.
func WithChatWaitable(ctx context.Context, cw ChatWaitable) context.Context {
	return context.WithValue(ctx, chatWaitableKey{}, cw)
}

func chatWaitableFromContext(ctx context.Context) (ChatWaitable, bool) {
	cw, ok := ctx.Value(chatWaitableKey{}).(ChatWaitable)
	return cw, ok
}

// RegisterRecv registers the chat-specific recvMessage op.
// Unlike messaging.New(), this op calls StartWaiting to signal the external
// chat server before suspending, which is required by the WaitMessage protocol.
func RegisterRecv(reg *ops.Registry) {
	reg.Register("recvMessage", &chatRecvMessageOp{})
}

type chatRecvMessageOp struct{}

func (chatRecvMessageOp) Kind() ops.OpKind { return ops.OpKindSuspendable }

type chatRecvMessageArgs struct {
	Set            string `json:"set"`
	Timeout        int    `json:"timeout"`
	MessageTimeout int    `json:"messageTimeout"`
	TimeoutSet     string `json:"timeoutSet"`
}

func (chatRecvMessageOp) Execute(ctx context.Context, in ops.OpInput) (ops.OpOutput, error) {
	var argv chatRecvMessageArgs
	if err := ops.DecodeArgs(in, &argv); err != nil {
		return ops.OpOutput{}, err
	}

	connID := messaging.ConnIDFromContext(ctx)
	suspendKey := "msg:" + connID

	// ── Resume path ──────────────────────────────────────────────────────────
	if in.ResumePayload != nil {
		if in.ResumePayload["timeout"] == "true" {
			out := ops.OpOutput{}
			if argv.TimeoutSet != "" {
				out.SetVars = map[string]string{argv.TimeoutSet: "true"}
			}
			return out, nil
		}

		msg := in.ResumePayload["msg"]

		if len(in.Triggers) > 0 && msg != "" {
			cmdKey := "commands-" + msg
			if trig, ok := in.Triggers[cmdKey]; ok {
				return ops.OpOutput{
					Branch:          trig,
					ReenterOnResume: true,
				}, nil
			}
		}

		out := ops.OpOutput{}
		if argv.Set != "" {
			out.SetVars = map[string]string{argv.Set: msg}
		}
		return out, nil
	}

	// ── Fresh path ────────────────────────────────────────────────────────────
	if connID == "" {
		return ops.OpOutput{}, fmt.Errorf("recvMessage: connection ID not in context")
	}

	// Signal the external chat server to forward the next inbound message.
	// Without this call, the chat server won't deliver the message to flow_manager.
	if cw, ok := chatWaitableFromContext(ctx); ok {
		cw.StartWaiting(argv.Timeout)
	}

	return ops.OpOutput{
		SuspendKey:      suspendKey,
		ReenterOnResume: true,
		Pending:         buildChatRecvPending(suspendKey, in.Node.ID, argv),
	}, nil
}

func buildChatRecvPending(suspendKey, nodeID string, argv chatRecvMessageArgs) *state.PendingIntent {
	args := map[string]string{}
	if argv.Timeout > 0 {
		// The timeout is handled by StartWaiting which fires fireInboundHandlers("")
		// after the timeout elapses. No wake_at needed here.
		args["timeout_sec"] = fmt.Sprintf("%d", argv.Timeout)
	}
	return &state.PendingIntent{
		OpName:    "recv_message",
		NodeID:    nodeID,
		ResumeKey: suspendKey,
		Args:      args,
	}
}
