package call

import (
	"context"
	"io"

	cc "buf.build/gen/go/webitel/cc/protocolbuffers/go"
	"github.com/webitel/flow_manager/flow"
	"github.com/webitel/flow_manager/model"
	"github.com/webitel/wlog"
)

type JoinAgentArgs struct {
	Agent *struct {
		Id        *int32  `json:"id"`
		Extension *string `json:"extension"`
	}
	Processing *struct {
		Enabled    bool
		RenewalSec uint32 `json:"renewal_sec"`
		Sec        uint32 `json:"sec"`
		Form       struct {
			Id   int32
			Name string
		} `json:"form"`
	}
	Ringtone         model.PlaybackFile `json:"ringtone"`
	Bridged          []interface{}      `json:"bridged"`
	Timeout          int32              `json:"timeout"`
	QueueName        string             `json:"queue_name"`
	CancelDistribute bool               `json:"cancel_distribute"`
}

func (r *Router) joinAgent(ctx context.Context, scope *flow.Flow, call model.Call, args interface{}) (model.Response, *model.AppError) {
	var argv JoinAgentArgs
	var agentId *int32

	if call.Direction() != model.CallDirectionInbound {
		// todo
		// error
	}

	if err := r.Decode(scope, args, &argv); err != nil {
		return nil, err
	}

	if argv.Agent == nil {
		return model.CallResponseError, ErrorRequiredParameter("joinAgent", "agent")
	}

	if argv.Agent.Id == nil && argv.Agent.Extension != nil {
		agentId, _ = r.fm.GetAgentIdByExtension(call.DomainId(), *argv.Agent.Extension)
	} else {
		agentId = argv.Agent.Id
	}

	if agentId == nil {
		return model.CallResponseError, ErrorRequiredParameter("joinAgent", "agent")
	}

	t := call.GetVariable("variable_transfer_history")
	var ringtone *cc.CallJoinToAgentRequest_WaitingMusic
	//FIXME
	if argv.Ringtone.Name != nil || argv.Ringtone.Id != nil {
		var err *model.AppError
		req := make([]*model.PlaybackFile, 1, 1)
		req[0] = &model.PlaybackFile{
			Id:   argv.Ringtone.Id,
			Name: argv.Ringtone.Name,
		}
		if req, err = r.fm.GetMediaFiles(call.DomainId(), &req); err != nil {
			return nil, err
		} else if req != nil && req[0] != nil && req[0].Type != nil {
			ringtone = &cc.CallJoinToAgentRequest_WaitingMusic{
				Id:   int32(*req[0].Id),
				Type: *req[0].Type,
			}
		}
	}

	req := &cc.CallJoinToAgentRequest{
		DomainId:         call.DomainId(),
		MemberCallId:     call.Id(),
		AgentId:          *agentId,
		WaitingMusic:     ringtone,
		Timeout:          argv.Timeout,
		Variables:        call.DumpExportVariables(),
		QueueName:        argv.QueueName,
		CancelDistribute: argv.CancelDistribute,
	}

	if argv.Processing != nil && argv.Processing.Enabled {
		req.Processing = &cc.CallJoinToAgentRequest_Processing{
			Enabled:    true,
			RenewalSec: argv.Processing.RenewalSec,
			Sec:        argv.Processing.Sec,
		}

		if argv.Processing.Form.Id > 0 {
			req.Processing.Form = &cc.QueueFormSchema{
				Id: argv.Processing.Form.Id,
			}
		}
	}

	res, err := r.fm.JoinToAgent(ctx, req)

	if err != nil {
		call.Log().Err(err)
		return model.CallResponseOK, nil
	}

	// TODO bug close stream channel
	for {
		var msg cc.QueueEvent
		err = res.RecvMsg(&msg)
		if err == io.EOF {
			break
		} else if err != nil {
			wlog.Error(err.Error())
			return model.CallResponseError, nil
		}

		switch e := msg.Data.(type) {
		case *cc.QueueEvent_Joined:
			call.Set(ctx, model.Variables{
				"attempt_id": e.Joined.AttemptId,
			})

		case *cc.QueueEvent_Bridged:
			if len(argv.Bridged) > 0 {
				call.Set(ctx, model.Variables{
					"agent_id":        e.Bridged.AgentId,
					"agent_extension": call.GetVariable("Caller-Caller-ID-Number"),
				})
				go flow.Route(ctx, scope.Fork("agent-bridged", flow.ArrInterfaceToArrayApplication(argv.Bridged)), r)
			}

		case *cc.QueueEvent_Leaving:
			call.Set(ctx, model.Variables{
				"cc_result": e.Leaving.Result,
			})
			break
		}
	}

	//call.Dump()

	if t != call.GetVariable("variable_transfer_history") {
		scope.SetCancel()
	}

	return model.CallResponseOK, nil
}
