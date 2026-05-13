package im

import (
	"context"
	"fmt"

	chatdomain "github.com/webitel/flow_manager/internal/domain/chat"
	"github.com/webitel/flow_manager/internal/runtime/ops"
)

// RegisterMenu registers the menu op.
func RegisterMenu(reg *ops.Registry) {
	reg.Register("menu", &menuOp{})
}

type menuOp struct{}

func (o *menuOp) Kind() ops.OpKind { return ops.OpKindSync }

func (o *menuOp) Execute(ctx context.Context, in ops.OpInput) (ops.OpOutput, error) {
	dialog, ok := dialogFromContext(ctx)
	if !ok {
		return ops.OpOutput{}, fmt.Errorf("menu: no IMDialog in context")
	}

	argv := chatdomain.ChatMenuArgs{Type: "buttons"}
	if err := ops.DecodeArgs(in, &argv); err != nil {
		return ops.OpOutput{}, err
	}

	switch argv.Type {
	case "buttons", "inline":
	default:
		return ops.OpOutput{}, nil
	}

	if _, appErr := dialog.SendMenu(ctx, &argv); appErr != nil {
		return ops.OpOutput{}, fmt.Errorf("menu: %s", appErr.Error())
	}
	return ops.OpOutput{}, nil
}
