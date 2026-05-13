// Package call — ops that require RouterDeps beyond model.Call.
package call

import (
	"context"
	"fmt"
	"net/http"

	genpb "github.com/webitel/flow_manager/api/gen/cc"
	apperrs "github.com/webitel/flow_manager/internal/infrastructure/errors"
	"github.com/webitel/flow_manager/internal/runtime/ops"
	"github.com/webitel/flow_manager/model"
)

// FMDeps is the narrow interface required by the FM call ops.
type FMDeps interface {
	SetCallGranteeId(domainId int64, id string, granteeId int64) error
	SetCallUserId(domainId int64, id string, userId int64) error
	UpdateCallFrom(id string, name, number, destination *string) error
	GetMediaFiles(domainId int64, req *[]*model.PlaybackFile) ([]*model.PlaybackFile, error)
	CallOutboundQueue(ctx context.Context, in *genpb.OutboundCallRequest) (*genpb.OutboundCallResponse, error)
}

// RegisterFM adds call ops that need FMDeps to reg.
func RegisterFM(reg *ops.Registry, deps FMDeps) {
	reg.Register("setGrantee", &setGranteeOp{deps: deps})
	reg.Register("setUser", &setUserOp{deps: deps})
	reg.Register("updateCid", &updateCidOp{deps: deps})
	reg.Register("ringback", &ringbackOp{deps: deps})
	reg.Register("backgroundPlayback", &backgroundPlaybackOp{deps: deps})
	reg.Register("ccOutbound", &ccOutboundOp{deps: deps})
}

// ── setGrantee ────────────────────────────────────────────────────────────────

type setGranteeOp struct{ deps FMDeps }

func (setGranteeOp) Kind() ops.OpKind { return ops.OpKindSync }

func (o *setGranteeOp) Execute(ctx context.Context, in ops.OpInput) (ops.OpOutput, error) {
	call, ok := callConnFromContext(ctx)
	if !ok {
		return ops.OpOutput{}, fmt.Errorf("setGrantee: no call connection in context")
	}
	var argv struct {
		Id int64 `json:"id"`
	}
	if err := ops.DecodeArgs(in, &argv); err != nil {
		return ops.OpOutput{}, err
	}
	if argv.Id < 1 {
		return ops.OpOutput{}, apperrs.New(http.StatusBadRequest, "setGrantee: id required")
	}
	if appErr := o.deps.SetCallGranteeId(call.DomainId(), call.Id(), argv.Id); appErr != nil {
		return ops.OpOutput{}, appErr
	}
	if _, appErr := call.Set(ctx, model.Variables{model.GranteeHeader: argv.Id}); appErr != nil {
		return ops.OpOutput{}, appErr
	}
	return ops.OpOutput{}, nil
}

// ── setUser ───────────────────────────────────────────────────────────────────

type setUserOp struct{ deps FMDeps }

func (setUserOp) Kind() ops.OpKind { return ops.OpKindSync }

func (o *setUserOp) Execute(ctx context.Context, in ops.OpInput) (ops.OpOutput, error) {
	call, ok := callConnFromContext(ctx)
	if !ok {
		return ops.OpOutput{}, fmt.Errorf("setUser: no call connection in context")
	}
	var argv struct {
		Id int64 `json:"id"`
	}
	if err := ops.DecodeArgs(in, &argv); err != nil {
		return ops.OpOutput{}, err
	}
	if argv.Id == 0 {
		return ops.OpOutput{}, apperrs.New(http.StatusBadRequest, "setUser: id required")
	}
	if appErr := o.deps.SetCallUserId(call.DomainId(), call.Id(), argv.Id); appErr != nil {
		return ops.OpOutput{}, appErr
	}
	return ops.OpOutput{}, nil
}

// ── updateCid ─────────────────────────────────────────────────────────────────

type updateCidOp struct{ deps FMDeps }

func (updateCidOp) Kind() ops.OpKind { return ops.OpKindSync }

func (o *updateCidOp) Execute(ctx context.Context, in ops.OpInput) (ops.OpOutput, error) {
	call, ok := callConnFromContext(ctx)
	if !ok {
		return ops.OpOutput{}, fmt.Errorf("updateCid: no call connection in context")
	}
	var argv struct {
		Name        string `json:"name"`
		Number      string `json:"number"`
		Destination string `json:"destination"`
	}
	if err := ops.DecodeArgs(in, &argv); err != nil {
		return ops.OpOutput{}, err
	}
	if argv.Name == "" && argv.Number == "" && argv.Destination == "" {
		return ops.OpOutput{}, apperrs.New(http.StatusBadRequest, "updateCid: name or number or destination required")
	}
	var name, number, destination *string
	if argv.Name != "" {
		name = &argv.Name
	}
	if argv.Number != "" {
		number = &argv.Number
	}
	if argv.Destination != "" {
		destination = &argv.Destination
	}
	if _, appErr := call.UpdateCid(ctx, name, number, destination); appErr != nil {
		return ops.OpOutput{}, appErr
	}
	if appErr := o.deps.UpdateCallFrom(call.Id(), name, number, destination); appErr != nil {
		return ops.OpOutput{}, appErr
	}
	return ops.OpOutput{}, nil
}

// ── ringback ──────────────────────────────────────────────────────────────────

type ringbackOp struct{ deps FMDeps }

func (ringbackOp) Kind() ops.OpKind { return ops.OpKindSync }

func (o *ringbackOp) Execute(ctx context.Context, in ops.OpInput) (ops.OpOutput, error) {
	call, ok := callConnFromContext(ctx)
	if !ok {
		return ops.OpOutput{}, fmt.Errorf("ringback: no call connection in context")
	}
	var argv struct {
		All      bool                `json:"all"`
		Call     *model.PlaybackFile `json:"call"`
		Hold     *model.PlaybackFile `json:"hold"`
		Transfer *model.PlaybackFile `json:"transfer"`
	}
	if err := ops.DecodeArgs(in, &argv); err != nil {
		return ops.OpOutput{}, err
	}
	search := []*model.PlaybackFile{argv.Call, argv.Hold, argv.Transfer}
	res, appErr := o.deps.GetMediaFiles(call.DomainId(), &search)
	if appErr != nil {
		return ops.OpOutput{}, appErr
	}
	if _, appErr = call.Ringback(ctx, argv.All, res[0], res[1], res[2]); appErr != nil {
		return ops.OpOutput{}, appErr
	}
	return ops.OpOutput{}, nil
}

// ── backgroundPlayback ────────────────────────────────────────────────────────

type backgroundPlaybackOp struct{ deps FMDeps }

func (backgroundPlaybackOp) Kind() ops.OpKind { return ops.OpKindSync }

func (o *backgroundPlaybackOp) Execute(ctx context.Context, in ops.OpInput) (ops.OpOutput, error) {
	call, ok := callConnFromContext(ctx)
	if !ok {
		return ops.OpOutput{}, fmt.Errorf("backgroundPlayback: no call connection in context")
	}
	var argv struct {
		Name            string              `json:"name"`
		File            *model.PlaybackFile `json:"file"`
		VolumeReduction int                 `json:"volumeReduction"`
	}
	if err := ops.DecodeArgs(in, &argv); err != nil {
		return ops.OpOutput{}, err
	}
	search := []*model.PlaybackFile{argv.File}
	res, appErr := o.deps.GetMediaFiles(call.DomainId(), &search)
	if appErr != nil {
		return ops.OpOutput{}, appErr
	}
	if _, appErr = call.BackgroundPlayback(ctx, res[0], argv.Name, argv.VolumeReduction); appErr != nil {
		return ops.OpOutput{}, appErr
	}
	return ops.OpOutput{}, nil
}

// ── ccOutbound ────────────────────────────────────────────────────────────────

type ccOutboundOp struct{ deps FMDeps }

func (ccOutboundOp) Kind() ops.OpKind { return ops.OpKindSync }

func (o *ccOutboundOp) Execute(ctx context.Context, in ops.OpInput) (ops.OpOutput, error) {
	call, ok := callConnFromContext(ctx)
	if !ok {
		return ops.OpOutput{}, fmt.Errorf("ccOutbound: no call connection in context")
	}
	if call.Direction() != model.CallDirectionOutbound {
		return ops.OpOutput{}, apperrs.New(http.StatusBadRequest, "call.cc_outbound: this call is not an outbound")
	}
	if call.UserId() == 0 {
		return ops.OpOutput{}, apperrs.New(http.StatusBadRequest, "call.cc_outbound: call originated from a non-user source")
	}
	var argv struct {
		QueueName        string                        `json:"queueName"`
		CancelDistribute bool                          `json:"cancelDistribute"`
		Processing       model.ProcessingWithoutAnswer `json:"processing"`
	}
	if err := ops.DecodeArgs(in, &argv); err != nil {
		return ops.OpOutput{}, err
	}
	vars := call.DumpExportVariables()
	if cid := call.GetContactId(); cid != 0 {
		vars["wbt_contact_id"] = fmt.Sprintf("%d", cid)
	}
	if call.Stopped() || call.HangupCause() != "" {
		return ops.OpOutput{}, nil
	}
	req := &genpb.OutboundCallRequest{
		CallId:    call.Id(),
		Timeout:   10,
		UserId:    int64(call.UserId()),
		DomainId:  call.DomainId(),
		Variables: vars,
		Processing: &genpb.OutboundCallRequest_Processing{
			Enabled:       argv.Processing.Enabled,
			RenewalSec:    argv.Processing.RenewalSec,
			Sec:           argv.Processing.Sec,
			Form:          &genpb.QueueFormSchema{Id: argv.Processing.Form.Id},
			WithoutAnswer: argv.Processing.WithoutAnswer,
		},
		QueueName:        argv.QueueName,
		CancelDistribute: argv.CancelDistribute,
	}
	if argv.Processing.Prolongation != nil && argv.Processing.Prolongation.Enabled {
		req.Processing.ProcessingProlongation = &genpb.ProcessingProlongation{
			Enabled:             argv.Processing.Prolongation.Enabled,
			RepeatsNumber:       argv.Processing.Prolongation.RepeatsNumber,
			ProlongationTimeSec: argv.Processing.Prolongation.ProlongationTimeSec,
			IsTimeoutRetry:      argv.Processing.Prolongation.IsTimeoutRetry,
		}
	}
	res, appErr := o.deps.CallOutboundQueue(ctx, req)
	if appErr != nil {
		return ops.OpOutput{}, nil // log and continue like legacy
	}
	return ops.OpOutput{
		SetVars: map[string]string{
			"cc_attempt_id": fmt.Sprintf("%d", res.AttemptId),
			"cc_agent_id":   fmt.Sprintf("%d", res.AttemptId),
		},
	}, nil
}
