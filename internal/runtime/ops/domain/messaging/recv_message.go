// Package messaging provides suspendable ops for inbound-message handling.
package messaging

import (
	"context"
	"fmt"
	"time"

	"github.com/webitel/flow_manager/internal/runtime/ops"
	"github.com/webitel/flow_manager/internal/runtime/state"
)

// connIDKey is the unexported context key used to pass the connection ID into
// the op without touching OpInput (which has no channel-specific fields).
type connIDKey struct{}

// WithConnID embeds connID into ctx so recvMessageOp can build its SuspendKey.
func WithConnID(ctx context.Context, connID string) context.Context {
	return context.WithValue(ctx, connIDKey{}, connID)
}

// ConnIDFromContext extracts the connection ID previously stored by WithConnID.
func ConnIDFromContext(ctx context.Context) string {
	v, _ := ctx.Value(connIDKey{}).(string)
	return v
}

// New returns the recv_message Op. No external dependencies are needed because
// the op only interacts with the execution state and the coordinator pattern.
func New() ops.Op { return recvMessageOp{} }

type recvMessageOp struct{}

func (recvMessageOp) Kind() ops.OpKind { return ops.OpKindSuspendable }

type recvMessageArgs struct {
	// Set is the variable name that will receive the inbound message text.
	// Named "set" to match the legacy recvMessage schema field.
	Set string `json:"set"`
	// Timeout is the maximum number of seconds to wait. 0 means wait forever.
	Timeout int `json:"timeout"`
	// TimeoutSet receives "true" when the wait expires without a message.
	TimeoutSet string `json:"timeoutSet"`
}

func (recvMessageOp) Execute(ctx context.Context, in ops.OpInput) (ops.OpOutput, error) {
	var argv recvMessageArgs
	if err := ops.DecodeArgs(in, &argv); err != nil {
		return ops.OpOutput{}, err
	}

	connID := ConnIDFromContext(ctx)
	suspendKey := "msg:" + connID

	// Resume path — op is called again because ReenterOnResume was set on the
	// initial suspend. Inspect the payload and either forward or re-suspend.
	if in.ResumePayload != nil {
		if in.ResumePayload["timeout"] == "true" {
			// Timer fired; set the timeout variable (if configured) and continue.
			out := ops.OpOutput{}
			if argv.TimeoutSet != "" {
				out.SetVars = map[string]string{argv.TimeoutSet: "true"}
			}
			return out, nil
		}

		msg := in.ResumePayload["msg"]

		// TriggerCommands: if message matches a declared command, run the
		// trigger sub-tree inline. ReenterOnResume backs up the position so
		// this op re-executes after the trigger finishes. The trigger may
		// itself contain a recvMessage — it suspends normally, frames are
		// persisted on the shared stack, and resumes arrive at the trigger's
		// op first. When the trigger completes the stack unwinds back here.
		if len(in.Triggers) > 0 {
			cmdKey := "commands-" + msg
			if trig, ok := in.Triggers[cmdKey]; ok {
				return ops.OpOutput{
					Branch:          trig,
					ReenterOnResume: true,
				}, nil
			}
		}

		// Plain message — set the target variable and continue.
		out := ops.OpOutput{}
		if argv.Set != "" {
			out.SetVars = map[string]string{argv.Set: msg}
		}
		return out, nil
	}

	// Initial suspend path.
	if connID == "" {
		return ops.OpOutput{}, fmt.Errorf("recv_message: connection ID not in context")
	}

	return ops.OpOutput{
		SuspendKey:      suspendKey,
		Pending:         buildPending(suspendKey, in.Node.ID, argv),
		ReenterOnResume: true,
	}, nil
}

func buildPending(suspendKey, nodeID string, argv recvMessageArgs) *state.PendingIntent {
	args := map[string]string{}
	if argv.Timeout > 0 {
		wakeAt := time.Now().Add(time.Duration(argv.Timeout) * time.Second)
		args["wake_at"] = wakeAt.UTC().Format(time.RFC3339)
	}
	return &state.PendingIntent{
		OpName:         "recv_message",
		NodeID:         nodeID,
		IdempotencyKey: suspendKey,
		Args:           args,
		ResumeKey:      suspendKey,
	}
}
