package email

import (
	"context"
	"fmt"

	"github.com/webitel/flow_manager/internal/runtime/ops"
	"github.com/webitel/flow_manager/internal/runtime/ops/connctx"
	"github.com/webitel/flow_manager/model"
)

// ReplyDeps is the subset of  the reply op needs.
type ReplyDeps interface {
	ReplyEmail(conn model.EmailConnection, text string) *model.AppError
}

// RegisterReply adds the reply op to reg.
func RegisterReply(reg *ops.Registry, deps ReplyDeps) {
	reg.Register("reply", &replyOp{deps: deps})
}

type replyOp struct{ deps ReplyDeps }

func (replyOp) Kind() ops.OpKind { return ops.OpKindSync }

type replyArgs struct {
	Body string `json:"body"`
}

func (o *replyOp) Execute(ctx context.Context, in ops.OpInput) (ops.OpOutput, error) {
	conn := connctx.ConnectionFromContext(ctx)
	if conn == nil {
		return ops.OpOutput{}, fmt.Errorf("reply: no connection in context")
	}
	emailConn, ok := conn.(model.EmailConnection)
	if !ok {
		return ops.OpOutput{}, fmt.Errorf("reply: connection is not an email connection")
	}

	var argv replyArgs
	if err := ops.DecodeArgs(in, &argv); err != nil {
		return ops.OpOutput{}, err
	}
	if argv.Body == "" {
		return ops.OpOutput{}, fmt.Errorf("reply: body is required")
	}

	if appErr := o.deps.ReplyEmail(emailConn, argv.Body); appErr != nil {
		return ops.OpOutput{}, fmt.Errorf("reply: %s", appErr.Error())
	}
	return ops.OpOutput{}, nil
}
