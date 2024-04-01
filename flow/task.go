package flow

import (
	"context"
	"github.com/webitel/flow_manager/model"
	"github.com/webitel/protos/cc"
	"github.com/webitel/wlog"
	"io"
)

type JoinAgentToTaskArgs struct {
	Agent *struct {
		Id        *int32  `json:"id"`
		Extension *string `json:"extension"`
	}
	Communication model.CallbackCommunication
	Processing    *struct {
		Enabled    bool
		RenewalSec uint32 `json:"renewal_sec"`
		Sec        uint32 `json:"sec"`
		Form       struct {
			Id   int
			Name string
		} `json:"form"`
	}
	Bridged          []interface{} `json:"bridged"`
	Timeout          int32         `json:"timeout"`
	QueueName        string        `json:"queue_name"`
	CancelDistribute bool          `json:"cancelDistribute"`
}

func (r *router) joinAgentToTask(ctx context.Context, scope *Flow, c model.Connection, args interface{}) (model.Response, *model.AppError) {
	var argv JoinAgentToTaskArgs
	var agentId *int32

	if err := scope.Decode(args, &argv); err != nil {
		return nil, err
	}

	if argv.Agent == nil {
		return model.CallResponseError, ErrorRequiredParameter("joinAgentToTask", "agent")
	}

	if argv.Agent.Id == nil && argv.Agent.Extension != nil {
		agentId, _ = r.fm.GetAgentIdByExtension(c.DomainId(), *argv.Agent.Extension)
	} else {
		agentId = argv.Agent.Id
	}

	if agentId == nil {
		return model.CallResponseError, ErrorRequiredParameter("joinAgentToTask", "agent")
	}

	req := &cc.TaskJoinToAgentRequest{
		DomainId: c.DomainId(),
		AgentId:  *agentId,
		Timeout:  argv.Timeout,
		//Variables:        c.DumpExportVariables(),
		QueueName:        argv.QueueName,
		CancelDistribute: argv.CancelDistribute,
		Destination: &cc.MemberCommunication{
			Destination: argv.Communication.Destination,
			Type:        &cc.MemberCommunicationType{},
		},
	}

	if argv.Communication.Type.Id != nil {
		req.Destination.Type.Id = int32(*argv.Communication.Type.Id)
	}
	if argv.Communication.Type.Name != nil {
		req.Destination.Type.Name = *argv.Communication.Type.Name
	}

	if argv.Communication.Description != nil {
		req.Destination.Destination = *argv.Communication.Description
	}

	if argv.Processing != nil && argv.Processing.Enabled {
		req.Processing = &cc.TaskJoinToAgentRequest_Processing{
			Enabled:    true,
			RenewalSec: argv.Processing.RenewalSec,
			Sec:        argv.Processing.Sec,
		}

		if argv.Processing.Form.Id > 0 {
			req.Processing.FormSchemaId = uint32(argv.Processing.Form.Id)
		}
	}

	res, err := r.fm.TaskJoinToAgent(ctx, req)

	if err != nil {
		wlog.Error(err.Error())
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
			c.Set(ctx, model.Variables{
				"attempt_id": e.Joined.AttemptId,
			})

		case *cc.QueueEvent_Bridged:
			if len(argv.Bridged) > 0 {
				c.Set(ctx, model.Variables{
					"agent_id": e.Bridged.AgentId,
				})
				//go Route(ctx, scope.Fork("agent-bridged", ArrInterfaceToArrayApplication(argv.Bridged)), r)
			}

		case *cc.QueueEvent_Leaving:
			c.Set(ctx, model.Variables{
				"cc_result": e.Leaving.Result,
			})
			break
		}
	}

	return ResponseOK, nil
}
