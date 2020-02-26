package flow

import (
	"fmt"
	"github.com/robertkrimen/otto"
	"github.com/webitel/flow_manager/model"
	"github.com/webitel/wlog"
	"net/http"
	"time"
)

type TimeFnList map[string]func(time.Time) string

var timeFnList TimeFnList

type conditionArgs struct {
	expression string
	then_      *Node
	else_      *Node
	vm_        *otto.Otto
	iter       *Flow
}

func newConditionArgs(i *Flow, parent *Node, props interface{}) *conditionArgs {
	args := &conditionArgs{
		then_: NewNode(parent),
		else_: NewNode(parent),
		iter:  i,
	}

	if tmp, ok := props.(map[string]interface{}); ok {
		if th, ok := tmp["then"].([]interface{}); ok {
			parseFlowArray(i, args.then_, ArrInterfaceToArrayApplication(th))
		}

		if el, ok := tmp["else"].([]interface{}); ok {
			parseFlowArray(i, args.else_, ArrInterfaceToArrayApplication(el))
		}

		//FIXME
		if ex, ok := tmp["sysExpression"].(string); ok {
			args.expression = ex
		}
	}

	return args
}

func (r *Router) conditionHandler(conn model.Connection, args interface{}) (model.Response, *model.AppError) {
	var req *conditionArgs
	var ok bool
	if req, ok = args.(*conditionArgs); !ok {
		return nil, model.NewAppError("Flow.ConditionHandler", "flow.condition_if.not_found", nil, "bad arguments", http.StatusBadRequest)
	}

	if req.vm_ == nil {
		req.vm_ = otto.New()
	}

	if value, err := req.vm_.Run(`_result = ` + req.expression); err == nil {
		if boolVal, err := value.ToBoolean(); err == nil && boolVal == true {
			wlog.Debug(fmt.Sprintf("condition (%s) = true", req.expression))
			req.iter.SetRoot(req.then_)
		} else {
			wlog.Debug(fmt.Sprintf("condition (%s) = false", req.expression))
			req.iter.SetRoot(req.else_)
		}
	} else {
		return nil, model.NewAppError("Flow.ConditionHandler", "flow.condition_if.vm_err", nil, err.Error(), http.StatusBadRequest)
	}

	return nil, nil
}
