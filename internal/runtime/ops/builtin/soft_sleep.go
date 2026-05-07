package builtin

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/webitel/flow_manager/internal/runtime/ops"
	"github.com/webitel/flow_manager/internal/runtime/state"
)

// softSleepArgs matches the schema format {"softSleep": "1000"} where the
// value is the sleep duration in milliseconds.
type softSleepArgs struct {
	SoftSleep int64 `json:"softSleep"`
}

type softSleepOp struct{}

// SoftSleep suspends the flow for a specified duration. It does not block the
// goroutine — instead it emits a SuspendKey so the Driver persists the state
// and exits. A timer worker (internal/adapters/inbound/im/timer.go) resumes the flow when the
// wake_at timestamp is reached.
func SoftSleep() ops.Op { return softSleepOp{} }

func (softSleepOp) Kind() ops.OpKind { return ops.OpKindSuspendable }

func (softSleepOp) Execute(_ context.Context, in ops.OpInput) (ops.OpOutput, error) {
	var args softSleepArgs
	if err := ops.DecodeArgs(in, &args); err != nil {
		return ops.OpOutput{}, fmt.Errorf("soft_sleep: decode args: %w", err)
	}

	d := time.Duration(args.SoftSleep) * time.Millisecond

	if d <= 0 {
		return ops.OpOutput{}, fmt.Errorf("soft_sleep: duration must be positive")
	}

	wakeAt := time.Now().UTC().Add(d)
	key := "soft_sleep:" + uuid.New().String()

	return ops.OpOutput{
		SuspendKey: key,
		Pending: &state.PendingIntent{
			OpName:         "soft_sleep",
			NodeID:         in.Node.ID,
			IdempotencyKey: key,
			Args:           map[string]string{"wake_at": wakeAt.Format(time.RFC3339)},
			ResumeKey:      key,
		},
	}, nil
}
