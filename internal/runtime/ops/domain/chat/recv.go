package chat

import (
	"context"
	"fmt"
	"strings"

	"github.com/webitel/flow_manager/internal/runtime/ops"
	"github.com/webitel/flow_manager/model"
)

// RegisterRecv registers the chat-specific recvMessage op.
// Chat uses the WaitMessage gRPC confirmation mechanism (blocking), not the
// OnInboundMessage / coordinator-suspend pattern used by IM.
func RegisterRecv(reg *ops.Registry) {
	reg.Register("recvMessage", &recvMessageOp{})
}

type recvMessageOp struct{}

func (recvMessageOp) Kind() ops.OpKind { return ops.OpKindSync }

type recvMessageArgs struct {
	Set            string `json:"set"`
	Timeout        int    `json:"timeout"`
	MessageTimeout int    `json:"messageTimeout"`
	Delimiter      string `json:"delimiter"`
}

func (recvMessageOp) Execute(ctx context.Context, in ops.OpInput) (ops.OpOutput, error) {
	conv, ok := conversationFromContext(ctx)
	if !ok {
		return ops.OpOutput{}, fmt.Errorf("recvMessage: no conversation in context")
	}

	var argv recvMessageArgs
	if err := ops.DecodeArgs(in, &argv); err != nil {
		return ops.OpOutput{}, err
	}

	if argv.Set == "" {
		return ops.OpOutput{}, nil
	}

	delimiter := " "
	if argv.Delimiter != "" {
		delimiter = argv.Delimiter
	}

	for {
		msgs, appErr := conv.ReceiveMessage(ctx, argv.Set, argv.Timeout, argv.MessageTimeout)
		if appErr != nil {
			_, _ = conv.Set(ctx, model.Variables{argv.Set: ""})
			return ops.OpOutput{SetVars: map[string]string{argv.Set: ""}}, fmt.Errorf("recvMessage: %s", appErr.Error())
		}

		// Check trigger commands; if matched, run the branch and wait again.
		if len(in.Triggers) > 0 && in.RunBranch != nil {
			matched := false
			for _, m := range msgs {
				cmdKey := "commands-" + m
				if branch, ok := in.Triggers[cmdKey]; ok {
					in.RunBranch(ctx, branch, in.Variables)
					matched = true
					break
				}
			}
			if matched {
				continue
			}
		}

		result := strings.Join(msgs, delimiter)
		_, _ = conv.Set(ctx, model.Variables{argv.Set: result})
		return ops.OpOutput{SetVars: map[string]string{argv.Set: result}}, nil
	}
}
