package call

import (
	"context"
	"fmt"
	"github.com/webitel/flow_manager/flow"
	"github.com/webitel/flow_manager/gen/cc"
	"github.com/webitel/flow_manager/model"
	"io"
)

type QueueOutbound struct {
	Name       string `json:"name"`
	Number     string `json:"number"`
	Processing struct {
		Enabled    bool
		RenewalSec uint32 `json:"renewal_sec"`
		Sec        uint32 `json:"sec"`
		Form       struct {
			Id   int32
			Name string
		} `json:"form"`
	}
}

func (r *Router) ccOutbound(ctx context.Context, scope *flow.Flow, call model.Call, args interface{}) (model.Response, *model.AppError) {
	var argv QueueOutbound
	if call.Direction() != model.CallDirectionOutbound {
		// error
	}

	if err := r.Decode(scope, args, &argv); err != nil {
		return nil, err
	}

	t := call.GetVariable("variable_transfer_history")
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

	res, err := r.fm.CallOutboundQueue(ctx, &cc.OutboundCallReqeust{
		CallId:    call.Id(),
		Timeout:   10,
		Variables: vars,
		Processing: &cc.OutboundCallReqeust_Processing{
			Enabled:    argv.Processing.Enabled,
			RenewalSec: argv.Processing.RenewalSec,
			Sec:        argv.Processing.Sec,
			Form: &cc.QueueFormSchema{
				Id: argv.Processing.Form.Id,
			},
		},
		QueueName:        "bla bla",
		CancelDistribute: false,
	})

	if err != nil {
		call.Log().Err(err)
		return model.CallResponseOK, nil
	}

	for {
		var msg cc.QueueEvent
		err = res.RecvMsg(&msg)
		if err == io.EOF {
			break
		} else if err != nil {
			call.Log().Error(err.Error())
			return model.CallResponseError, nil
		}

		switch e := msg.Data.(type) {

		case *cc.QueueEvent_Leaving:
			call.Set(ctx, model.Variables{
				"cc_result": e.Leaving.Result,
			})
			break
		default:
			call.Log().Error(fmt.Sprintf("unexpected type %T", msg.Data))
		}
	}

	if t != call.GetVariable("variable_transfer_history") {
		scope.SetCancel()
	}

	return model.CallResponseOK, nil
}
