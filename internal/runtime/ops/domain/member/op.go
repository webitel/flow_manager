// Package member provides native ops for CC member/queue operations:
// ccPosition, memberInfo, patchMembers, ewt, callbackQueue.
package member

import (
	"context"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/webitel/flow_manager/internal/domain/flow"
	"github.com/webitel/flow_manager/internal/domain/queue"
	"github.com/webitel/flow_manager/internal/runtime/ops"
	"github.com/webitel/flow_manager/store"
)

// New registers all member ops on reg using the provided MemberStore.
func Register(reg *ops.Registry, s store.MemberStore) {
	reg.Register("ccPosition", &ccPositionOp{store: s})
	reg.Register("memberInfo", &memberInfoOp{store: s})
	reg.Register("patchMembers", &patchMembersOp{store: s})
	reg.Register("ewt", &ewtOp{store: s})
	reg.Register("callbackQueue", &callbackQueueOp{store: s})
}

// ── ccPosition ────────────────────────────────────────────────────────────────

type ccPositionOp struct{ store store.MemberStore }

func (o *ccPositionOp) Kind() ops.OpKind { return ops.OpKindSync }

type ccPositionArgs struct {
	Set string `json:"set"`
}

func (o *ccPositionOp) Execute(ctx context.Context, in ops.OpInput) (ops.OpOutput, error) {
	var argv ccPositionArgs
	if err := ops.DecodeArgs(in, &argv); err != nil {
		return ops.OpOutput{}, err
	}
	if argv.Set == "" {
		return ops.OpOutput{}, fmt.Errorf("ccPosition: set is required")
	}
	pos, err := o.store.CallPosition(in.ConnID)
	if err != nil {
		return ops.OpOutput{}, fmt.Errorf("ccPosition: %w", err)
	}
	return ops.OpOutput{SetVars: map[string]string{argv.Set: fmt.Sprintf("%d", pos)}}, nil
}

// ── memberInfo ────────────────────────────────────────────────────────────────

type memberInfoOp struct{ store store.MemberStore }

func (o *memberInfoOp) Kind() ops.OpKind { return ops.OpKindSync }

type memberInfoArgs struct {
	Member *queue.SearchMember `json:"member"`
	Set    flow.Variables     `json:"set"`
}

func (o *memberInfoOp) Execute(ctx context.Context, in ops.OpInput) (ops.OpOutput, error) {
	var argv memberInfoArgs
	if err := ops.DecodeArgs(in, &argv); err != nil {
		return ops.OpOutput{}, err
	}
	if argv.Member == nil {
		return ops.OpOutput{}, fmt.Errorf("memberInfo: member is required")
	}
	if len(argv.Set) == 0 {
		return ops.OpOutput{}, fmt.Errorf("memberInfo: set is required")
	}
	res, err := o.store.GetProperties(in.DomainID, argv.Member, argv.Set)
	if err != nil {
		return ops.OpOutput{}, fmt.Errorf("memberInfo: %w", err)
	}
	setVars := make(map[string]string, len(res))
	for k, v := range res {
		setVars[k] = fmt.Sprintf("%v", v)
	}
	return ops.OpOutput{SetVars: setVars}, nil
}

// ── patchMembers ──────────────────────────────────────────────────────────────

type patchMembersOp struct{ store store.MemberStore }

func (o *patchMembersOp) Kind() ops.OpKind { return ops.OpKindSync }

type patchMembersArgs struct {
	Member *queue.SearchMember `json:"member"`
	Patch  *queue.PatchMember  `json:"patch"`
}

func (o *patchMembersOp) Execute(ctx context.Context, in ops.OpInput) (ops.OpOutput, error) {
	var argv patchMembersArgs
	if err := ops.DecodeArgs(in, &argv); err != nil {
		return ops.OpOutput{}, err
	}
	if argv.Member == nil {
		return ops.OpOutput{}, fmt.Errorf("patchMembers: member is required")
	}
	if argv.Patch == nil {
		return ops.OpOutput{}, fmt.Errorf("patchMembers: patch is required")
	}
	if argv.Patch.StopCauseDep != nil && argv.Patch.StopCause == nil {
		argv.Patch.StopCause = argv.Patch.StopCauseDep
	}
	if argv.Patch.ReadyAtDep != nil && argv.Patch.ReadyAt == nil {
		argv.Patch.ReadyAt = argv.Patch.ReadyAtDep
	}
	if _, err := o.store.PatchMembers(in.DomainID, argv.Member, argv.Patch); err != nil {
		return ops.OpOutput{}, fmt.Errorf("patchMembers: %w", err)
	}
	return ops.OpOutput{}, nil
}

// ── ewt ───────────────────────────────────────────────────────────────────────

type ewtOp struct{ store store.MemberStore }

func (o *ewtOp) Kind() ops.OpKind { return ops.OpKindSync }

type ewtArgs struct {
	SetVar    string `json:"setVar"`
	QueueIds  []int  `json:"queue_ids"`
	BucketIds []int  `json:"bucket_ids"`
	Strategy  string `json:"strategy"`
	Min       int    `json:"min"`
}

func (o *ewtOp) Execute(ctx context.Context, in ops.OpInput) (ops.OpOutput, error) {
	var argv ewtArgs
	if err := ops.DecodeArgs(in, &argv); err != nil {
		return ops.OpOutput{}, err
	}
	if argv.SetVar == "" {
		return ops.OpOutput{}, fmt.Errorf("ewt: setVar is required")
	}
	if argv.Min == 0 {
		argv.Min = 60
	}
	val, err := o.store.EWTPuzzle(in.DomainID, in.ConnID, argv.Min, argv.QueueIds, argv.BucketIds)
	if err != nil {
		return ops.OpOutput{}, fmt.Errorf("ewt: %w", err)
	}
	return ops.OpOutput{SetVars: map[string]string{argv.SetVar: fmt.Sprintf("%f", val)}}, nil
}

// ── callbackQueue ─────────────────────────────────────────────────────────────

type callbackQueueOp struct{ store store.MemberStore }

func (o *callbackQueueOp) Kind() ops.OpKind { return ops.OpKindSync }

type callbackQueueParams struct {
	QueueId int `json:"queue_id"`
	HoldSec int `json:"holdSec"`
}

func (o *callbackQueueOp) Execute(ctx context.Context, in ops.OpInput) (ops.OpOutput, error) {
	var params callbackQueueParams
	if err := ops.DecodeArgs(in, &params); err != nil {
		return ops.OpOutput{}, fmt.Errorf("callbackQueue: %w", err)
	}
	var member queue.CallbackMember
	if err := ops.DecodeArgs(in, &member); err != nil {
		return ops.OpOutput{}, fmt.Errorf("callbackQueue: %w", err)
	}

	// deprecated queue_id — queue.id takes precedence
	if member.Queue.Id != nil {
		params.QueueId = *member.Queue.Id
	}

	// deprecated communication.type_id
	if member.Communication.TypeId != nil && member.Communication.Type.Id == nil {
		member.Communication.Type.Id = member.Communication.TypeId
	}

	if member.StopCause != nil && *member.StopCause == "" {
		member.StopCause = nil
	}

	if member.Name != "" && !utf8.ValidString(member.Name) {
		member.Name = strings.ToValidUTF8(member.Name, "")
	}

	if err := o.store.CreateMember(in.DomainID, params.QueueId, params.HoldSec, &member); err != nil {
		return ops.OpOutput{}, fmt.Errorf("callbackQueue: %w", err)
	}
	return ops.OpOutput{}, nil
}
