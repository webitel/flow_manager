package builtin_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/webitel/flow_manager/internal/runtime/ops"
	"github.com/webitel/flow_manager/internal/runtime/ops/builtin"
	"github.com/webitel/flow_manager/internal/runtime/tree"
)

func jsInput(data, setVar string, vars map[string]string, globalVar func(string) string) ops.OpInput {
	return ops.OpInput{
		Node: &tree.Node{Args: map[string]any{
			"data":   data,
			"setVar": setVar,
		}},
		Variables: vars,
		GlobalVar:  globalVar,
	}
}

func runJs(t *testing.T, data, setVar string, vars map[string]string, globalVar func(string) string) (ops.OpOutput, error) {
	t.Helper()
	return builtin.JsOp().Execute(context.Background(), jsInput(data, setVar, vars, globalVar))
}

// --- correctness ---

func TestJsOp_Arithmetic(t *testing.T) {
	out, err := runJs(t, `return 6 * 7`, "result", nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if out.SetVars["result"] != "42" {
		t.Errorf("want 42, got %q", out.SetVars["result"])
	}
}

func TestJsOp_StringLiteral(t *testing.T) {
	out, err := runJs(t, `return "hello"`, "r", nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if out.SetVars["r"] != "hello" {
		t.Errorf("want %q, got %q", "hello", out.SetVars["r"])
	}
}

func TestJsOp_ChannelVar(t *testing.T) {
	out, err := runJs(t, `return ${name} + " world"`, "r",
		map[string]string{"name": "hello"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if out.SetVars["r"] != "hello world" {
		t.Errorf("want %q, got %q", "hello world", out.SetVars["r"])
	}
}

func TestJsOp_GlobalVar(t *testing.T) {
	globals := map[string]string{"env": "prod"}
	out, err := runJs(t, `return $${env}`, "r", nil, func(k string) string { return globals[k] })
	if err != nil {
		t.Fatal(err)
	}
	if out.SetVars["r"] != "prod" {
		t.Errorf("want %q, got %q", "prod", out.SetVars["r"])
	}
}

func TestJsOp_ChannelVarAndGlobalVar(t *testing.T) {
	globals := map[string]string{"prefix": "PRE"}
	out, err := runJs(t, `return $${prefix} + "-" + ${suffix}`, "r",
		map[string]string{"suffix": "SUF"},
		func(k string) string { return globals[k] })
	if err != nil {
		t.Fatal(err)
	}
	if out.SetVars["r"] != "PRE-SUF" {
		t.Errorf("want %q, got %q", "PRE-SUF", out.SetVars["r"])
	}
}

func TestJsOp_LocalDateIsFunction(t *testing.T) {
	out, err := runJs(t, `return typeof LocalDate`, "r", nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if out.SetVars["r"] != "function" {
		t.Errorf("want %q, got %q", "function", out.SetVars["r"])
	}
}

func TestJsOp_UndefinedResult_NoSetVars(t *testing.T) {
	out, err := runJs(t, `var x = 1`, "r", nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(out.SetVars) != 0 {
		t.Errorf("expected no SetVars, got %v", out.SetVars)
	}
}

func TestJsOp_NullResult_NoSetVars(t *testing.T) {
	out, err := runJs(t, `return null`, "r", nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(out.SetVars) != 0 {
		t.Errorf("expected no SetVars for null, got %v", out.SetVars)
	}
}

func TestJsOp_EmptySetVar_NoSetVars(t *testing.T) {
	out, err := runJs(t, `return 99`, "", nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(out.SetVars) != 0 {
		t.Errorf("expected no SetVars when setVar is empty, got %v", out.SetVars)
	}
}

func TestJsOp_RuntimeError(t *testing.T) {
	_, err := runJs(t, `null.foo`, "r", nil, nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestJsOp_SyntaxError(t *testing.T) {
	_, err := runJs(t, `return (((`, "r", nil, nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestJsOp_Timeout(t *testing.T) {
	if testing.Short() {
		t.Skip("timeout test takes ~1s")
	}
	_, err := runJs(t, `while(true){}`, "r", nil, nil)
	if err == nil {
		t.Fatal("expected timeout error, got nil")
	}
	if !strings.Contains(err.Error(), "timeout") && !errors.Is(err, context.DeadlineExceeded) {
		t.Logf("error: %v", err)
	}
}

// --- benchmarks ---

func BenchmarkJsOp_Arithmetic(b *testing.B) {
	op := builtin.JsOp()
	in := jsInput(`return 6 * 7`, "r", nil, nil)
	ctx := context.Background()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := op.Execute(ctx, in); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkJsOp_VarLookup(b *testing.B) {
	op := builtin.JsOp()
	in := jsInput(`return ${x} + ${y}`, "r",
		map[string]string{"x": "hello", "y": " world"}, nil)
	ctx := context.Background()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := op.Execute(ctx, in); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkJsOp_StringManipulation(b *testing.B) {
	op := builtin.JsOp()
	in := jsInput(`return ${val}.toLowerCase().trim().replace(/\s+/g, "_")`, "r",
		map[string]string{"val": "  Hello World  "}, nil)
	ctx := context.Background()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := op.Execute(ctx, in); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkJsOp_VMCreationOnly(b *testing.B) {
	// isolates how much of the cost is VM construction vs. script execution
	vars := map[string]string{"k": "v"}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		builtin.BuildJsVMExported(vars, nil, "")
	}
}
