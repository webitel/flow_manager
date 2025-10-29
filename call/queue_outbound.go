package call

import (
	"context"
	"fmt"

	"github.com/webitel/flow_manager/flow"
	"github.com/webitel/flow_manager/gen/cc"
	"github.com/webitel/flow_manager/model"
	"github.com/webitel/wlog"
)

type QueueOutbound struct {
	Name             string                        `json:"name"`
	Number           string                        `json:"number"`
	QueueName        string                        `json:"queueName"`
	CancelDistribute bool                          `json:"cancelDistribute"`
	Processing       model.ProcessingWithoutAnswer `json:"processing"`
}

func (r *Router) ccOutbound(ctx context.Context, scope *flow.Flow, call model.Call, args any) (model.Response, *model.AppError) {
	var argv QueueOutbound
	if call.Direction() != model.CallDirectionOutbound {
		return nil, model.NewRequestError("call.cc_outbound", "this call is not an outbound")
	}

	if call.UserId() == 0 {
		return nil, model.NewRequestError("call.cc_outbound", "call originated from a non-user source")
	}

	if err := r.Decode(scope, args, &argv); err != nil {
		return nil, err
	}

	vars := call.DumpExportVariables()

	if cid := call.GetContactId(); cid != 0 {
		vars["wbt_contact_id"] = fmt.Sprintf("%d", cid)
	}

	if call.Stopped() {
		return model.CallResponseError, nil
	}

	if call.HangupCause() != "" {
		return nil, model.NewAppError("Call", "call.cc_put.join.hangup", nil, "Call is down", 500)
	}

	ocr := &cc.OutboundCallRequest{
		CallId:    call.Id(),
		Timeout:   10,
		UserId:    int64(call.UserId()),
		DomainId:  call.DomainId(),
		Variables: vars,
		Processing: &cc.OutboundCallRequest_Processing{
			Enabled:    argv.Processing.Enabled,
			RenewalSec: argv.Processing.RenewalSec,
			Sec:        argv.Processing.Sec,
			Form: &cc.QueueFormSchema{
				Id: argv.Processing.Form.Id,
			},
			WithoutAnswer: argv.Processing.WithoutAnswer,
		},
		QueueName:        argv.QueueName,
		CancelDistribute: argv.CancelDistribute,
	}

	if argv.Processing.Prolongation != nil && argv.Processing.Prolongation.Enabled {
		ocr.Processing.ProcessingProlongation = &cc.ProcessingProlongation{
			Enabled:             argv.Processing.Prolongation.Enabled,
			RepeatsNumber:       argv.Processing.Prolongation.RepeatsNumber,
			ProlongationTimeSec: argv.Processing.Prolongation.ProlongationTimeSec,
			IsTimeoutRetry:      argv.Processing.Prolongation.IsTimeoutRetry,
		}
	}

	res, err := r.fm.CallOutboundQueue(ctx, ocr)

	if err != nil {
		call.Log().Err(err)
		return model.CallResponseOK, nil
	}

	call.Log().Debug("accept outbound queue call",
		wlog.Int64("attempt_id", res.AttemptId),
	)

	return call.Set(ctx, model.Variables{
		"cc_attempt_id": res.AttemptId,
		"cc_gent_id":    res.AttemptId,
	})
}
