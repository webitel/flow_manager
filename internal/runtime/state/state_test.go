package state_test

import (
	"encoding/json"
	"testing"

	"github.com/webitel/flow_manager/internal/runtime/state"
)

func TestExecState_JSONRoundTrip(t *testing.T) {
	tags := map[string]string{"retry": "1.2", "end": "5"}
	s := state.NewExecState(42, 0xdeadbeef, tags)
	s.Variables["greeting"] = "hello"
	s.Variables["count"] = "3"
	s.Stack = append(s.Stack, state.Frame{NodeID: "1.0", Position: 2})
	s.GotoCounter = 7

	b, err := json.Marshal(s)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var got state.ExecState
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if got.SchemaID != s.SchemaID {
		t.Errorf("SchemaID: got %d, want %d", got.SchemaID, s.SchemaID)
	}
	if got.SchemaVersion != s.SchemaVersion {
		t.Errorf("SchemaVersion: got %d, want %d", got.SchemaVersion, s.SchemaVersion)
	}
	if got.Status != state.StatusRunning {
		t.Errorf("Status: got %q, want %q", got.Status, state.StatusRunning)
	}
	if len(got.Stack) != 2 {
		t.Errorf("Stack len: got %d, want 2", len(got.Stack))
	} else {
		if got.Stack[0].NodeID != "root" || got.Stack[0].Position != 0 {
			t.Errorf("Stack[0]: got %+v", got.Stack[0])
		}
		if got.Stack[1].NodeID != "1.0" || got.Stack[1].Position != 2 {
			t.Errorf("Stack[1]: got %+v", got.Stack[1])
		}
	}
	if got.Variables["greeting"] != "hello" || got.Variables["count"] != "3" {
		t.Errorf("Variables: got %v", got.Variables)
	}
	if got.Tags["retry"] != "1.2" || got.Tags["end"] != "5" {
		t.Errorf("Tags: got %v", got.Tags)
	}
	if got.GotoCounter != 7 {
		t.Errorf("GotoCounter: got %d, want 7", got.GotoCounter)
	}
	if got.Pending != nil {
		t.Errorf("Pending should be nil")
	}
}

func TestExecState_WithPending_JSONRoundTrip(t *testing.T) {
	s := state.NewExecState(10, 1, nil)
	s.Pending = &state.PendingIntent{
		OpName:         "joinQueue",
		NodeID:         "3",
		IdempotencyKey: "abc-123",
		Args:           map[string]string{"queue_id": "77"},
		ResumeKey:      "queue:77:abc-123",
	}
	s.Status = state.StatusSuspended

	b, err := json.Marshal(s)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var got state.ExecState
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if got.Status != state.StatusSuspended {
		t.Errorf("Status: got %q", got.Status)
	}
	if got.Pending == nil {
		t.Fatal("Pending is nil after round-trip")
	}
	if got.Pending.OpName != "joinQueue" {
		t.Errorf("Pending.OpName: got %q", got.Pending.OpName)
	}
	if got.Pending.ResumeKey != "queue:77:abc-123" {
		t.Errorf("Pending.ResumeKey: got %q", got.Pending.ResumeKey)
	}
	if got.Pending.Args["queue_id"] != "77" {
		t.Errorf("Pending.Args: got %v", got.Pending.Args)
	}
}

func TestExecState_EmptyVariables_JSONRoundTrip(t *testing.T) {
	s := state.NewExecState(1, 0, nil)

	b, err := json.Marshal(s)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var got state.ExecState
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if got.Variables == nil {
		t.Error("Variables must not be nil after round-trip")
	}
	if len(got.Stack) != 1 || got.Stack[0].NodeID != "root" {
		t.Errorf("Stack: got %v", got.Stack)
	}
}
