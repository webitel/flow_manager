package builtin

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/dop251/goja"

	"github.com/webitel/flow_manager/internal/runtime/ops"
)

var errJsTimeout = errors.New("js: execution timeout")

type jsOp struct{}

// JsOp returns the native js op: evaluates a JavaScript snippet and stores
// the result in a schema variable.
func JsOp() ops.Op { return jsOp{} }

func (jsOp) Kind() ops.OpKind { return ops.OpKindSync }

func (o jsOp) Execute(_ context.Context, in ops.OpInput) (ops.OpOutput, error) {
	data, _ := in.Node.Args["data"].(string)
	setVar, _ := in.Node.Args["setVar"].(string)

	// rewrite ${var} → _getChannelVar("var"), $${var} → _getGlobalVar("var")
	data = reExprGlobalVar.ReplaceAllStringFunc(data, func(s string) string {
		m := reExprGlobalVar.FindStringSubmatch(s)
		return fmt.Sprintf(`_getGlobalVar("%s")`, m[1])
	})
	data = reExprVar.ReplaceAllStringFunc(data, func(s string) string {
		m := reExprVar.FindStringSubmatch(s)
		return fmt.Sprintf(`_getChannelVar("%s")`, m[1])
	})

	vm := buildJsVM(in.Variables, in.GlobalVar, in.Timezone)

	type result struct {
		val goja.Value
		err error
	}
	ch := make(chan result, 1)

	go func() {
		v, err := vm.RunString(`
var LocalDate = function() {
	var t = _LocalDateParameters();
	return new Date(t[0], t[1]-1, t[2], t[3], t[4], t[5]);
};
(function(LocalDate) {` + data + `})(LocalDate)`)
		ch <- result{v, err}
	}()

	timer := time.AfterFunc(1*time.Second, func() {
		vm.Interrupt(errJsTimeout)
	})
	defer timer.Stop()

	r := <-ch
	if r.err != nil {
		return ops.OpOutput{}, fmt.Errorf("js: %w", r.err)
	}

	if setVar == "" || r.val == nil || goja.IsUndefined(r.val) || goja.IsNull(r.val) {
		return ops.OpOutput{}, nil
	}
	return ops.OpOutput{SetVars: map[string]string{setVar: r.val.String()}}, nil
}

// buildJsVM creates a goja VM with _getChannelVar, _getGlobalVar, _LocalDateParameters.
func buildJsVM(vars map[string]string, globalVar func(string) string, timezone string) *goja.Runtime {
	vm := goja.New()

	vm.Set("_getChannelVar", func(call goja.FunctionCall) goja.Value {
		key := call.Argument(0).String()
		return vm.ToValue(vars[key])
	})

	vm.Set("_getGlobalVar", func(call goja.FunctionCall) goja.Value {
		if globalVar == nil {
			return vm.ToValue("")
		}
		return vm.ToValue(globalVar(call.Argument(0).String()))
	})

	now := time.Now()
	if timezone != "" {
		if loc, err := time.LoadLocation(timezone); err == nil {
			now = now.In(loc)
		}
	}
	vm.Set("_LocalDateParameters", func(call goja.FunctionCall) goja.Value {
		return vm.ToValue([]int{now.Year(), int(now.Month()), now.Day(), now.Hour(), now.Minute(), now.Second()})
	})

	return vm
}
