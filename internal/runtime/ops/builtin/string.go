package builtin

import (
	"context"
	"crypto/md5"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/base64"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/dop251/goja"

	"github.com/webitel/flow_manager/internal/runtime/ops"
)

// stringArgs matches the schema format:
//
//	{"string": {"setVar": "out", "fn": "toUpperCase", "data": "hello", "args": []}}
type stringArgs struct {
	SetVar string        `json:"setVar"`
	Fn     string        `json:"fn"`
	Data   string        `json:"data"`
	Args   []interface{} `json:"args"`
}

type stringOp struct{}

// StringOp implements the "string" builtin op.
func StringOp() ops.Op { return stringOp{} }

func (stringOp) Kind() ops.OpKind { return ops.OpKindSync }

func (stringOp) Execute(_ context.Context, in ops.OpInput) (ops.OpOutput, error) {
	var args stringArgs
	if err := ops.DecodeArgs(in, &args); err != nil {
		return ops.OpOutput{}, fmt.Errorf("string: decode args: %w", err)
	}
	if args.SetVar == "" {
		return ops.OpOutput{}, fmt.Errorf("string: setVar is required")
	}
	if args.Fn == "" {
		return ops.OpOutput{}, fmt.Errorf("string: fn is required")
	}

	var value string
	switch args.Fn {
	case "reverse":
		value = strReverse(args.Data)
	case "charAt":
		value = strCharAt(args.Data, strTopIntArg(args.Args))
	case "base64":
		value = strBase64(strTopArg(args.Args), args.Data)
	case "MD5":
		value = fmt.Sprintf("%x", md5.Sum([]byte(args.Data)))
	case "SHA-256":
		value = fmt.Sprintf("%x", sha256.Sum256([]byte(args.Data)))
	case "SHA-512":
		value = fmt.Sprintf("%x", sha512.Sum512([]byte(args.Data)))
	case "length":
		value = fmt.Sprintf("%d", len([]rune(args.Data)))
	case "gomatch":
		value = strGoMatch(args.Data, strTopArg(args.Args))
	default:
		v, err := strRunJS(args.Fn, args.Data, args.Args)
		if err != nil {
			return ops.OpOutput{}, fmt.Errorf("string: fn %q: %w", args.Fn, err)
		}
		value = v
	}

	return ops.OpOutput{SetVars: map[string]string{args.SetVar: value}}, nil
}

func strRunJS(fn, data string, fnArgs []interface{}) (string, error) {
	vm := goja.New()
	_ = vm.Set("fnName", fn)
	_ = vm.Set("data", data)
	_ = vm.Set("args", fnArgs)
	v, err := vm.RunString(`
		var value;
		var mappedArgs;
		if (Array.isArray(args)) {
			mappedArgs = args.map(function(v) {
				if (typeof v === "string") {
					var m = v.match(/^\/(.+)\/([gimy]*)$/);
					if (m) { return new RegExp(m[1], m[2]); }
				}
				return v;
			});
		} else {
			mappedArgs = args !== undefined && args !== null ? [args] : [];
		}
		if (typeof data[fnName] === "function") {
			value = data[fnName].apply(data, mappedArgs);
		} else {
			throw new Error("unknown string function: " + fnName);
		}
		value !== undefined && value !== null ? String(value) : "";
	`)
	if err != nil {
		return "", err
	}
	return v.String(), nil
}

func strReverse(s string) string {
	r := []rune(s)
	for i, j := 0, len(r)-1; i < j; i, j = i+1, j-1 {
		r[i], r[j] = r[j], r[i]
	}
	return string(r)
}

func strCharAt(s string, pos int) string {
	r := []rune(s)
	if pos >= 0 && pos < len(r) {
		return string(r[pos])
	}
	return ""
}

func strBase64(mode, data string) string {
	switch mode {
	case "encoder":
		return base64.StdEncoding.EncodeToString([]byte(data))
	case "decoder":
		b, _ := base64.StdEncoding.DecodeString(data)
		return string(b)
	}
	return ""
}

func strGoMatch(s, expr string) string {
	r, err := regexp.Compile(expr)
	if err != nil {
		return ""
	}
	return strings.Join(r.FindStringSubmatch(s), ",")
}

func strTopArg(args []interface{}) string {
	if len(args) > 0 {
		return fmt.Sprintf("%v", args[0])
	}
	return ""
}

func strTopIntArg(args []interface{}) int {
	n, _ := strconv.Atoi(strTopArg(args))
	return n
}
