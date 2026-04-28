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
// When fn is "random" (or omitted), data is the pool of values to pick from.
//
// Supported fn values:
//
//	Go-native:
//	  random  — pick a uniformly random element from data array (Webitel custom)
//
//	JS-native (delegated to goja via Math.*):
//	  round   — Math.round(data[0])
//	  floor   — Math.floor(data[0])
//	  ceil    — Math.ceil(data[0])
//	  abs     — Math.abs(data[0])
//	  max     — Math.max(...data)
//	  min     — Math.min(...data)
//	  pow     — Math.pow(data[0], data[1])
//	  sqrt    — Math.sqrt(data[0])
//	  PI      — Math.PI constant (data ignored)
type mathArgs struct {
	SetVar string        `json:"setVar"`
	Fn     string        `json:"fn"`
	Data   []interface{} `json:"data"`
}

type mathOp struct{}

// MathOp returns the "math" builtin op.
//
// Schema examples:
//
//	{"math": {"setVar": "greeting", "fn": "random", "data": ["Hello!", "Hi there!", "Welcome!"]}}
//	{"math": {"setVar": "rounded",  "fn": "round",  "data": ["${raw_score}"]}}
//	{"math": {"setVar": "biggest",  "fn": "max",    "data": [1, 5, 3]}}
//	{"math": {"setVar": "pi_val",   "fn": "PI"}}
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
