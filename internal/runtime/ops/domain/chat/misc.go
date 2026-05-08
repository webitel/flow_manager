package chat

import (
	"context"
	"fmt"

	"github.com/webitel/flow_manager/internal/runtime/ops"
	"github.com/webitel/flow_manager/model"
)

// RegisterMisc registers bridge, export, menu, unSet, and recvMessage.
// All ops retrieve the conversation from context — no external deps needed.
func RegisterMisc(reg *ops.Registry) {
	reg.Register("bridge", &bridgeOp{})
	reg.Register("export", &exportOp{})
	reg.Register("menu", &menuOp{})
	reg.Register("unSet", &unSetOp{})
}

// ── bridge ────────────────────────────────────────────────────────────────────

type bridgeOp struct{}

func (bridgeOp) Kind() ops.OpKind { return ops.OpKindSync }

type bridgeArgs struct {
	UserId  int64 `json:"userId"`
	Timeout int   `json:"timeout"`
}

func (bridgeOp) Execute(ctx context.Context, in ops.OpInput) (ops.OpOutput, error) {
	conv, ok := conversationFromContext(ctx)
	if !ok {
		return ops.OpOutput{}, fmt.Errorf("bridge: no conversation in context")
	}
	var argv bridgeArgs
	if err := ops.DecodeArgs(in, &argv); err != nil {
		return ops.OpOutput{}, err
	}
	if appErr := conv.Bridge(ctx, argv.UserId, argv.Timeout); appErr != nil {
		return ops.OpOutput{}, fmt.Errorf("bridge: %s", appErr.Error())
	}
	return ops.OpOutput{}, nil
}

// ── export ────────────────────────────────────────────────────────────────────

type exportOp struct{}

func (exportOp) Kind() ops.OpKind { return ops.OpKindSync }

func (exportOp) Execute(ctx context.Context, in ops.OpInput) (ops.OpOutput, error) {
	conv, ok := conversationFromContext(ctx)
	if !ok {
		return ops.OpOutput{}, fmt.Errorf("export: no conversation in context")
	}
	keys := rawStringSlice(in)
	if _, appErr := conv.Export(ctx, keys); appErr != nil {
		return ops.OpOutput{}, fmt.Errorf("export: %s", appErr.Error())
	}
	return ops.OpOutput{}, nil
}

// ── menu ──────────────────────────────────────────────────────────────────────

type menuOp struct{}

func (menuOp) Kind() ops.OpKind { return ops.OpKindSync }

func (menuOp) Execute(ctx context.Context, in ops.OpInput) (ops.OpOutput, error) {
	conv, ok := conversationFromContext(ctx)
	if !ok {
		return ops.OpOutput{}, fmt.Errorf("menu: no conversation in context")
	}
	var argv model.ChatMenuArgs
	argv.Type = "buttons"
	if err := ops.DecodeArgs(in, &argv); err != nil {
		return ops.OpOutput{}, err
	}
	switch argv.Type {
	case "buttons", "inline":
	default:
		return ops.OpOutput{}, fmt.Errorf("menu: unsupported type %q", argv.Type)
	}
	if _, appErr := conv.SendMenu(ctx, &argv); appErr != nil {
		return ops.OpOutput{}, fmt.Errorf("menu: %s", appErr.Error())
	}
	return ops.OpOutput{}, nil
}

// ── unSet ─────────────────────────────────────────────────────────────────────

type unSetOp struct{}

func (unSetOp) Kind() ops.OpKind { return ops.OpKindSync }

func (unSetOp) Execute(ctx context.Context, in ops.OpInput) (ops.OpOutput, error) {
	conv, ok := conversationFromContext(ctx)
	if !ok {
		return ops.OpOutput{}, fmt.Errorf("unSet: no conversation in context")
	}
	keys := rawStringSlice(in)
	if len(keys) == 0 {
		return ops.OpOutput{}, fmt.Errorf("unSet: at least one variable name required")
	}
	if _, appErr := conv.UnSet(ctx, keys); appErr != nil {
		return ops.OpOutput{}, fmt.Errorf("unSet: %s", appErr.Error())
	}
	return ops.OpOutput{}, nil
}
