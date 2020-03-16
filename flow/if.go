package flow

import (
	"fmt"
	"github.com/robertkrimen/otto"
	"github.com/webitel/flow_manager/model"
	"github.com/webitel/wlog"
	"net/http"
	"regexp"
)

var (
	compileProtectedFunctions *regexp.Regexp
	compileVars               *regexp.Regexp
	compileFunctions          *regexp.Regexp
)

func init() {
	compileProtectedFunctions = regexp.MustCompile(`\b(function|case|if|return|new|switch|var|this|typeof|for|while|break|do|continue)\b`)
	compileVars = regexp.MustCompile(`\$\{([\s\S]*?)\}`)
	compileFunctions = regexp.MustCompile(`\&(year|yday|mon|mday|week|mweek|wday|hour|minute|minute_of_day|time_of_day|date_time)\(([\s\S]*?)\)`)
}

type conditionArgs struct {
	expression string
	then_      *Node
	else_      *Node
	vm_        *otto.Otto
	flow       *Flow
}

func newConditionArgs(i *Flow, parent *Node, props interface{}) *conditionArgs {
	args := &conditionArgs{
		then_: NewNode(parent),
		else_: NewNode(parent),
		flow:  i,
	}

	if tmp, ok := props.(map[string]interface{}); ok {
		if th, ok := tmp["then"].([]interface{}); ok {
			parseFlowArray(i, args.then_, ArrInterfaceToArrayApplication(th))
		}

		if el, ok := tmp["else"].([]interface{}); ok {
			parseFlowArray(i, args.else_, ArrInterfaceToArrayApplication(el))
		}

		if ex, ok := tmp["expression"].(string); ok {
			args.expression = parseExpression(ex)
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

	injectJsSysObject(conn, req)

	if value, err := req.vm_.Run(`_result = ` + conn.ParseText(req.expression)); err == nil {
		if boolVal, err := value.ToBoolean(); err == nil && boolVal == true {
			wlog.Debug(fmt.Sprintf("condition (%s) is true", req.expression))
			req.flow.SetRoot(req.then_)
		} else {
			wlog.Debug(fmt.Sprintf("condition (%s) is false", req.expression))
			req.flow.SetRoot(req.else_)
		}
	} else {
		return nil, model.NewAppError("Flow.ConditionHandler", "flow.condition_if.vm_err", nil, err.Error(), http.StatusBadRequest)
	}

	return ResponseOK, nil
}

func parseExpression(expression string) string {

	expression = compileVars.ReplaceAllStringFunc(expression, func(varName string) string {
		l := compileVars.FindStringSubmatch(varName)
		return fmt.Sprintf(`sys.getVariable("%s")`, l[1])
	})

	expression = compileFunctions.ReplaceAllStringFunc(expression, func(s string) string {
		l := compileFunctions.FindStringSubmatch(s)

		return fmt.Sprintf(`sys.%s("%s")`, l[1], l[2])
	})

	expression = compileProtectedFunctions.ReplaceAllLiteralString(expression, "")

	return expression
}

func injectJsSysObject(conn model.Connection, args *conditionArgs) *otto.Object {
	sys, _ := args.vm_.Object("sys = {}")
	sys.Set("getVariable", func(call otto.FunctionCall) otto.Value {
		val, _ := conn.Get(call.Argument(0).String())
		res, err := args.vm_.ToValue(val)
		if err != nil {
			return otto.Value{}
		}
		return res
	})
	return sys
}
