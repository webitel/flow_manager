package notification

import (
	"context"
	"testing"

	"github.com/webitel/flow_manager/internal/domain/notification"
	"github.com/webitel/flow_manager/internal/runtime/ops"
	"github.com/webitel/flow_manager/internal/runtime/tree"
)

// ── fakeDeps ──────────────────────────────────────────────────────────────────

type fakeDeps struct {
	calls []notification.Notification
}

func (f *fakeDeps) UserNotification(n notification.Notification) {
	f.calls = append(f.calls, n)
}

var _ Deps = (*fakeDeps)(nil)

// ── helpers ───────────────────────────────────────────────────────────────────

func notifInput(args map[string]any) ops.OpInput {
	return ops.OpInput{Node: &tree.Node{Args: args}, DomainID: 7}
}

// ── tests ─────────────────────────────────────────────────────────────────────

func TestNotification_FireAndForget(t *testing.T) {
	deps := &fakeDeps{}
	op := New(deps)
	_, err := op.Execute(context.Background(), notifInput(map[string]any{
		"userIds": []any{int64(1), int64(2)},
		"message": "Hello operators!",
		"timeout": 5000,
		"type":    "info",
	}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(deps.calls) != 1 {
		t.Fatalf("UserNotification called %d times, want 1", len(deps.calls))
	}
	n := deps.calls[0]
	if n.DomainId != 7 {
		t.Errorf("DomainId = %d, want 7", n.DomainId)
	}
	if n.Action != notificationAction {
		t.Errorf("Action = %q, want %q", n.Action, notificationAction)
	}
	body, ok := n.Body.(map[string]interface{})
	if !ok {
		t.Fatalf("Body type = %T, want map[string]interface{}", n.Body)
	}
	if body["message"] != "Hello operators!" {
		t.Errorf("Body[message] = %v, want Hello operators!", body["message"])
	}
}

func TestNotification_EmptyArgs_NoError(t *testing.T) {
	// No validation — fires even with no args.
	deps := &fakeDeps{}
	op := New(deps)
	_, err := op.Execute(context.Background(), notifInput(map[string]any{}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(deps.calls) != 1 {
		t.Error("expected UserNotification to be called")
	}
}

func TestNotification_CreatedAtSet(t *testing.T) {
	deps := &fakeDeps{}
	op := New(deps)
	op.Execute(context.Background(), notifInput(map[string]any{"message": "test"})) //nolint:errcheck
	if deps.calls[0].CreatedAt == 0 {
		t.Error("expected CreatedAt to be set")
	}
}
