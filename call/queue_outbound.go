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
	Name             string `json:"name"`
	Number           string `json:"number"`
	QueueName        string `json:"queueName"`
	CancelDistribute bool   `json:"cancelDistribute"`
	Processing       struct {
		Enabled    bool
		RenewalSec uint32 `json:"renewalSec"`
		Sec        uint32 `json:"sec"`
		Form       struct {
			Id   int32
			Name string
		} `json:"form"`
		WithoutAnswer bool `json:"withoutAnswer"`
	}
}

func (r *Router) ccOutbound(ctx context.Context, scope *flow.Flow, call model.Call, args interface{}) (model.Response, *model.AppError) {
	var argv QueueOutbound
	if call.Direction() != model.CallDirectionOutbound {
		return nil, model.NewRequestError("call.cc_outbound", "this call is not an outbound")
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

	res, err := r.fm.CallOutboundQueue(ctx, &cc.OutboundCallRequest{
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
	})

	if err != nil {
		call.Log().Err(err)
		return model.CallResponseOK, nil
	}

	call.Log().Debug("accept outbound queue call",
		wlog.Int64("attempt_id", res.AttemptId),
	)

	return model.CallResponseOK, nil
}
