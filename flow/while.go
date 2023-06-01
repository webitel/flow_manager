package flow

import (
	"context"
	"fmt"
	"net/http"
	"strconv"

	"github.com/robertkrimen/otto"
	"github.com/webitel/flow_manager/model"
	"github.com/webitel/wlog"
)

type whileArgs struct {
	condition string
	do        *Node
	flow      *Flow
	// step is a local counter of cycle [while]
	step int
	// counter is a general counter of current node [while]
	counter int
	// max steps is a general limit for current node [while]
	maxSteps       int
	originalParent *ApplicationRequest
}

func newWhileArgs(i *Flow, parent *Node, req ApplicationRequest) *whileArgs {
	props := req.args
	args := &whileArgs{
		do:       NewNode(parent),
		flow:     i,
		maxSteps: 1000,
	}

	if tmp, ok := props.(map[string]interface{}); ok {
		if el, ok := tmp["do"].([]interface{}); ok {
			parseFlowArray(i, args.do, ArrInterfaceToArrayApplication(el))
		}
		if ex, ok := tmp["condition"].(string); ok {
			args.condition = parseExpression(ex)
		}
		if ex, ok := tmp["maxSteps"].(string); ok {
			if v, err := strconv.Atoi(ex); err == nil {
				args.maxSteps = v
			}
		}
		req.args = args

		if len(parent.children) == 0 {
			args.originalParent = nil
		} else {
			args.originalParent = parent.children[len(parent.children)-1]
		}

		args.do.Add(req)
	}

	return args
}

func (r *router) whileHandler(ctx context.Context, scope *Flow, conn model.Connection, args interface{}) (model.Response, *model.AppError) {

	var req *whileArgs
	var ok bool

	if req, ok = args.(*whileArgs); !ok {
		return nil, model.NewAppError("Flow.whileHandler", "flow.while.not_found", nil, "bad arguments", http.StatusBadRequest)
	}

	if req.counter != 0 && scope.getPreviousRequest() == req.originalParent { // called from [goto], reset steps count
		endCycle(req)
	}

	if req.counter < req.maxSteps { // execute cycle
		vm := scope.GetVm()
		injectJsSysObject(conn, vm, req.flow)
		// first step of cycle - execute [do] without conditions
		if req.step == 0 || checkCondition(req.condition, vm, conn) {
			nextStep(req)
		} else { // error!
			endCycle(req)
		}
		req.counter++
	}

	return ResponseOK, nil
}
func nextStep(req *whileArgs) {
	if len(req.do.children) == 1 { // nothing to do. == 1 because there will be always inserted [while] at the end
		endCycle(req)
		return
	}
	req.do.setFirst()
	req.flow.SetRoot(req.do)
	req.step++

}

func endCycle(req *whileArgs) {
	req.step = 0
}

func checkCondition(condition string, vm *otto.Otto, conn model.Connection) bool {
	var res bool
	if value, err := vm.Run(`_result = ` + conn.ParseText(condition)); err == nil { // else check condition
		res, err = value.ToBoolean()
		if err != nil {
			wlog.Debug(fmt.Sprintf("error while converting the result of condition (%s)", err.Error()))
		}
	} else {
		wlog.Debug(fmt.Sprintf("error while resolving condition (%s)", condition))
	}
	return res
}
