package calendar

import (
	"context"
	"fmt"
	"testing"

	"github.com/webitel/flow_manager/internal/runtime/ops"
	"github.com/webitel/flow_manager/internal/runtime/tree"
)

func calInput(args map[string]any) ops.OpInput {
	return ops.OpInput{Node: &tree.Node{Args: args}, DomainID: 1}
}

func makeCheckFn(res *Result, err error) CheckFn {
	return func(_ context.Context, _ int64, _ *int, _ *string) (*Result, error) {
		return res, err
	}
}

func TestCalendar_EmptySetVar_ReturnsEarly(t *testing.T) {
	op := New(makeCheckFn(&Result{Accept: true}, nil))
	out, err := op.Execute(context.Background(), calInput(map[string]any{}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(out.SetVars) != 0 {
		t.Errorf("expected no SetVars when setVar is empty, got %v", out.SetVars)
	}
}

func TestCalendar_Accept_SetsTrue(t *testing.T) {
	op := New(makeCheckFn(&Result{Accept: true}, nil))
	out, err := op.Execute(context.Background(), calInput(map[string]any{"setVar": "is_open"}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.SetVars["is_open"] != "true" {
		t.Errorf("got %q, want true", out.SetVars["is_open"])
	}
}

func TestCalendar_Reject_SetsFalse(t *testing.T) {
	op := New(makeCheckFn(&Result{Accept: false}, nil))
	out, err := op.Execute(context.Background(), calInput(map[string]any{"setVar": "is_open"}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.SetVars["is_open"] != "false" {
		t.Errorf("got %q, want false", out.SetVars["is_open"])
	}
}

func TestCalendar_Expire_Basic_SetsFalse(t *testing.T) {
	// Without extended=true, Expire → "false"
	op := New(makeCheckFn(&Result{Expire: true}, nil))
	out, err := op.Execute(context.Background(), calInput(map[string]any{"setVar": "is_open"}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.SetVars["is_open"] != "false" {
		t.Errorf("got %q, want false", out.SetVars["is_open"])
	}
}

func TestCalendar_Expire_Extended_SetsExpire(t *testing.T) {
	op := New(makeCheckFn(&Result{Expire: true}, nil))
	out, err := op.Execute(context.Background(), calInput(map[string]any{
		"setVar": "cal_result", "extended": true,
	}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.SetVars["cal_result"] != "expire" {
		t.Errorf("got %q, want expire", out.SetVars["cal_result"])
	}
}

func TestCalendar_Exception_Extended_SetsExceptionName(t *testing.T) {
	name := "Christmas"
	op := New(makeCheckFn(&Result{Excepted: &name}, nil))
	out, err := op.Execute(context.Background(), calInput(map[string]any{
		"setVar": "cal_result", "extended": true,
	}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.SetVars["cal_result"] != "Christmas" {
		t.Errorf("got %q, want Christmas", out.SetVars["cal_result"])
	}
}

func TestCalendar_DepError_Propagated(t *testing.T) {
	op := New(makeCheckFn(nil, fmt.Errorf("calendar service unavailable")))
	_, err := op.Execute(context.Background(), calInput(map[string]any{"setVar": "is_open"}))
	if err == nil {
		t.Fatal("expected error when CheckFn fails")
	}
}
