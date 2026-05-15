package messaging

import (
	"context"
	"testing"

	"github.com/webitel/flow_manager/internal/runtime/ops"
	"github.com/webitel/flow_manager/internal/runtime/tree"
)

func recvInput(args map[string]any) ops.OpInput {
	return ops.OpInput{Node: &tree.Node{Args: args}}
}

func recvInputWithPayload(args map[string]any, payload map[string]string) ops.OpInput {
	return ops.OpInput{Node: &tree.Node{Args: args}, ResumePayload: payload}
}

// ── fresh path ────────────────────────────────────────────────────────────────

func TestRecvMessage_Fresh_NoConnID(t *testing.T) {
	op := New()
	_, err := op.Execute(context.Background(), recvInput(map[string]any{"set": "reply"}))
	if err == nil {
		t.Fatal("expected error when connID is not in context")
	}
}

func TestRecvMessage_Fresh_Suspends(t *testing.T) {
	ctx := WithConnID(context.Background(), "conn-1")
	op := New()
	out, err := op.Execute(ctx, recvInput(map[string]any{"set": "reply", "timeout": 60}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.SuspendKey != "msg:conn-1" {
		t.Errorf("SuspendKey = %q, want msg:conn-1", out.SuspendKey)
	}
	if !out.ReenterOnResume {
		t.Error("expected ReenterOnResume=true")
	}
	if out.Pending == nil {
		t.Error("expected Pending to be set")
	}
}

func TestRecvMessage_Fresh_WithTimeout_HasWakeAt(t *testing.T) {
	ctx := WithConnID(context.Background(), "conn-2")
	op := New()
	out, err := op.Execute(ctx, recvInput(map[string]any{"timeout": 30}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Pending == nil {
		t.Fatal("expected Pending")
	}
	if out.Pending.Args["wake_at"] == "" {
		t.Error("expected wake_at to be set when timeout > 0")
	}
}

func TestRecvMessage_Fresh_NoTimeout_NoWakeAt(t *testing.T) {
	ctx := WithConnID(context.Background(), "conn-3")
	op := New()
	out, err := op.Execute(ctx, recvInput(map[string]any{"set": "msg"}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Pending != nil && out.Pending.Args["wake_at"] != "" {
		t.Error("expected no wake_at when timeout == 0")
	}
}

// ── resume path ───────────────────────────────────────────────────────────────

func TestRecvMessage_Resume_Timeout_SetsTimeoutVar(t *testing.T) {
	op := New()
	in := recvInputWithPayload(
		map[string]any{"set": "reply", "timeoutSet": "timed_out"},
		map[string]string{"timeout": "true"},
	)
	out, err := op.Execute(context.Background(), in)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.SetVars["timed_out"] != "true" {
		t.Errorf("timed_out = %q, want true", out.SetVars["timed_out"])
	}
}

func TestRecvMessage_Resume_Timeout_NoTimeoutSet(t *testing.T) {
	// timeoutSet not configured → empty SetVars, no error
	op := New()
	in := recvInputWithPayload(
		map[string]any{"set": "reply"},
		map[string]string{"timeout": "true"},
	)
	out, err := op.Execute(context.Background(), in)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(out.SetVars) != 0 {
		t.Errorf("expected empty SetVars, got %v", out.SetVars)
	}
}

func TestRecvMessage_Resume_Message_SetsVar(t *testing.T) {
	op := New()
	in := recvInputWithPayload(
		map[string]any{"set": "user_reply"},
		map[string]string{"msg": "hello"},
	)
	out, err := op.Execute(context.Background(), in)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.SetVars["user_reply"] != "hello" {
		t.Errorf("user_reply = %q, want hello", out.SetVars["user_reply"])
	}
}

func TestRecvMessage_Resume_TriggerMatch(t *testing.T) {
	trigNode := &tree.Node{ID: "cancel-branch"}
	op := New()
	in := ops.OpInput{
		Node:          &tree.Node{Args: map[string]any{"set": "msg"}},
		ResumePayload: map[string]string{"msg": "/cancel"},
		Triggers:      map[string]*tree.Node{"commands-/cancel": trigNode},
	}
	out, err := op.Execute(context.Background(), in)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Branch != trigNode {
		t.Error("expected trigger branch to be returned")
	}
	if !out.ReenterOnResume {
		t.Error("expected ReenterOnResume after trigger dispatch")
	}
}

func TestRecvMessage_Resume_NoSet_EmptyOutput(t *testing.T) {
	// When argv.Set == "" the message is consumed silently.
	op := New()
	in := recvInputWithPayload(
		map[string]any{},
		map[string]string{"msg": "anything"},
	)
	out, err := op.Execute(context.Background(), in)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(out.SetVars) != 0 {
		t.Errorf("expected empty SetVars, got %v", out.SetVars)
	}
}
