package builtin

import (
	"context"
	"fmt"
	"math/rand"

	"github.com/dop251/goja"

	"github.com/webitel/flow_manager/internal/runtime/ops"
)

// mathArgs matches the schema format:
//
//	{"math": {"setVar": "out", "fn": "round", "data": [3.7]}}
//
// "data" holds the arguments passed to the math function.
// When fn is "random", data is the pool of values to pick from.
type mathArgs struct {
	SetVar string        `json:"setVar"`
	Fn     string        `json:"fn"`
	Data   []interface{} `json:"data"`
}

type mathOp struct{}

// MathOp implements the "math" builtin op.
func MathOp() ops.Op { return mathOp{} }

func (mathOp) Kind() ops.OpKind { return ops.OpKindSync }

func (mathOp) Execute(_ context.Context, in ops.OpInput) (ops.OpOutput, error) {
	args := mathArgs{Fn: "random"}
	if err := ops.DecodeArgs(in, &args); err != nil {
		return ops.OpOutput{}, fmt.Errorf("math: decode args: %w", err)
	}
	if args.SetVar == "" {
		return ops.OpOutput{}, fmt.Errorf("math: setVar is required")
	}

	var value string
	switch args.Fn {
	case "random", "":
		if len(args.Data) == 0 {
			value = ""
		} else {
			value = fmt.Sprintf("%v", args.Data[rand.Intn(len(args.Data))])
		}
	default:
		v, err := mathRunJS(args.Fn, args.Data)
		if err != nil {
			return ops.OpOutput{}, fmt.Errorf("math: fn %q: %w", args.Fn, err)
		}
		value = v
	}

	return ops.OpOutput{SetVars: map[string]string{args.SetVar: value}}, nil
}

func mathRunJS(fn string, data []interface{}) (string, error) {
	vm := goja.New()
	_ = vm.Set("fnName", fn)
	_ = vm.Set("args", data)
	v, err := vm.RunString(`
		var value;
		if (typeof Math[fnName] === "function") {
			value = Math[fnName].apply(null, args);
		} else if (Math.hasOwnProperty(fnName)) {
			value = Math[fnName];
		} else {
			throw new Error("unknown math function: " + fnName);
		}
		if (typeof value === "number" && isNaN(value)) { value = ""; }
		value !== undefined && value !== null ? String(value) : "";
	`)
	if err != nil {
		return "", err
	}
	return v.String(), nil
}
