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
	if connID == "" {
		return ops.OpOutput{}, fmt.Errorf("recv_message: connection ID not in context")
	}

	suspendKey := "msg:" + connID

	args := map[string]string{}
	varFromPayload := map[string]string{}

	if argv.Set != "" {
		varFromPayload["msg"] = argv.Set
	}
	if argv.Timeout > 0 {
		wakeAt := time.Now().Add(time.Duration(argv.Timeout) * time.Second)
		args["wake_at"] = wakeAt.UTC().Format(time.RFC3339)
		if argv.TimeoutSet != "" {
			varFromPayload["timeout"] = argv.TimeoutSet
		}
	}

	return ops.OpOutput{
		SuspendKey: suspendKey,
		Pending: &state.PendingIntent{
			OpName:         "recv_message",
			NodeID:         in.Node.ID,
			IdempotencyKey: suspendKey,
			Args:           args,
			ResumeKey:      suspendKey,
			VarFromPayload: varFromPayload,
		},
	}, nil
}
