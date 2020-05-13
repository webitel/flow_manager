package flow

import (
	"context"
	"fmt"
	"github.com/webitel/flow_manager/model"
	"github.com/webitel/wlog"
	"net/http"
)

type SwitchArgs struct {
	Variable string
	Cases    map[string]*Node
	Flow     *Flow
}

func (s *SwitchArgs) String() string {
	return fmt.Sprintf("variable = %v", s.Variable)
}

func newSwitchArgs(i *Flow, parent *Node, props interface{}) (*SwitchArgs, *model.AppError) {
	var ok bool
	var tmp map[string]interface{}
	var cases map[string]interface{}

	args := &SwitchArgs{
		Flow:  i,
		Cases: make(map[string]*Node),
	}

	if tmp, ok = props.(map[string]interface{}); ok {
		if args.Variable, ok = tmp["variable"].(string); !ok || args.Variable == "" {
			return nil, model.NewAppError("Iterator", "iterator.parse_app.switch.valid_name", nil, "bad arguments, variable is required", http.StatusBadRequest)
		}

		if cases, ok = tmp["case"].(map[string]interface{}); !ok {
			return nil, model.NewAppError("Iterator", "iterator.parse_app.switch.valid_name", nil, "bad arguments, case is required", http.StatusBadRequest)
		}

		for caseName, caseVal := range cases {
			if c, ok := caseVal.([]interface{}); ok {
				args.Cases[caseName] = NewNode(parent)
				parseFlowArray(i, args.Cases[caseName], ArrInterfaceToArrayApplication(c))
			}
		}
	}

	return args, nil
}

func (r *router) switchHandler(ctx context.Context, scope *Flow, conn model.Connection, args interface{}) (model.Response, *model.AppError) {
	var req *SwitchArgs
	var ok bool
	var newNode *Node

	if req, ok = args.(*SwitchArgs); !ok {
		return nil, model.NewAppError("Flow.SwitchHandler", "flow.condition_switch.not_found", nil, "bad arguments", http.StatusBadRequest)
	}

	if newNode, ok = req.Cases[conn.ParseText(req.Variable)]; ok {
		newNode.setFirst()
		req.Flow.SetRoot(newNode)
		wlog.Debug(fmt.Sprintf("[%s] set switch case: %s", conn.Id(), req.Variable))
	} else if newNode, ok = req.Cases["default"]; ok {
		newNode.setFirst()
		req.Flow.SetRoot(newNode)
		wlog.Debug(fmt.Sprintf("call %s set switch default case %s", conn.Id(), req.Variable))
	}

	return ResponseOK, nil
}
