package flow

import (
	"github.com/webitel/flow_manager/model"
	"net/http"
)

type switchArgs struct {
	variable string
	cases    map[string]*Node
}

func newSwitchArgs(i *Flow, parent *Node, props interface{}) (*switchArgs, *model.AppError) {
	var ok bool
	var tmp map[string]interface{}
	var cases map[string]interface{}

	args := &switchArgs{
		cases: make(map[string]*Node),
	}

	if tmp, ok = props.(map[string]interface{}); ok {
		if args.variable, ok = tmp["variable"].(string); !ok || args.variable == "" {
			return nil, model.NewAppError("Iterator", "iterator.parse_app.switch.valid_name", nil, "bad arguments, variable is required", http.StatusBadRequest)
		}

		if cases, ok = tmp["case"].(map[string]interface{}); !ok {
			return nil, model.NewAppError("Iterator", "iterator.parse_app.switch.valid_name", nil, "bad arguments, case is required", http.StatusBadRequest)
		}

		for caseName, caseVal := range cases {
			if c, ok := caseVal.([]interface{}); ok {
				args.cases[caseName] = NewNode(parent)
				parseFlowArray(i, args.cases[caseName], ArrInterfaceToArrayApplication(c))
			}
		}
	}

	return args, nil
}
