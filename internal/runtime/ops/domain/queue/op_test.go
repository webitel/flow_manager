package queue

import (
	"fmt"
	"testing"

	"github.com/webitel/flow_manager/internal/domain/flow"
	queuedomain "github.com/webitel/flow_manager/internal/domain/queue"
	"github.com/webitel/flow_manager/internal/runtime/ops"
	"github.com/webitel/flow_manager/internal/runtime/tree"
	"github.com/webitel/flow_manager/internal/storage"

	"context"
)

// ── fakeQueueStore ────────────────────────────────────────────────────────────

type fakeQueueStore struct {
	historyResult float64
	historyErr    error
	queueData     flow.Variables
	queueDataErr  error
	agentsData    flow.Variables
	agentsErr     error
}

func (f *fakeQueueStore) HistoryStatistics(_ int64, _ *queuedomain.SearchQueueCompleteStatistics) (float64, error) {
	return f.historyResult, f.historyErr
}
func (f *fakeQueueStore) GetQueueData(_ int64, _ *queuedomain.SearchEntity, _ flow.Variables) (flow.Variables, error) {
	return f.queueData, f.queueDataErr
}
func (f *fakeQueueStore) GetQueueAgents(_ int64, _ int, _ string, _ flow.Variables) (flow.Variables, error) {
	return f.agentsData, f.agentsErr
}
func (f *fakeQueueStore) FindQueueByName(_ int64, _ string) (int32, error) {
	return 0, nil
}

var _ storage.QueueStore = (*fakeQueueStore)(nil)

// ── helpers ───────────────────────────────────────────────────────────────────

func queueInput(args map[string]any) ops.OpInput {
	return ops.OpInput{Node: &tree.Node{Args: args}, DomainID: 1}
}

// ── getQueueMetrics ───────────────────────────────────────────────────────────

func TestGetQueueMetrics_NoQueue(t *testing.T) {
	op := &getQueueMetricsOp{store: &fakeQueueStore{}}
	_, err := op.Execute(context.Background(), queueInput(map[string]any{"set": "out"}))
	if err == nil {
		t.Fatal("expected error when queue is nil")
	}
}

func TestGetQueueMetrics_NoSet(t *testing.T) {
	op := &getQueueMetricsOp{store: &fakeQueueStore{}}
	_, err := op.Execute(context.Background(), queueInput(map[string]any{
		"queue": map[string]any{"id": 1},
	}))
	if err == nil {
		t.Fatal("expected error when set is empty")
	}
}

func TestGetQueueMetrics_NonComplete_ReturnsZero(t *testing.T) {
	// When calls != "complete" the op returns 0 without hitting the store.
	op := &getQueueMetricsOp{store: &fakeQueueStore{historyResult: 99}}
	out, err := op.Execute(context.Background(), queueInput(map[string]any{
		"queue": map[string]any{"id": 1},
		"set":   "wait_time",
		"calls": "active",
	}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.SetVars["wait_time"] != "0" {
		t.Errorf("got %q, want 0", out.SetVars["wait_time"])
	}
}

func TestGetQueueMetrics_Complete_CallsStore(t *testing.T) {
	store := &fakeQueueStore{historyResult: 42.5}
	op := &getQueueMetricsOp{store: store}
	out, err := op.Execute(context.Background(), queueInput(map[string]any{
		"queue":  map[string]any{"id": 1},
		"set":    "avg_wait",
		"calls":  "complete",
		"metric": "avg",
	}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.SetVars["avg_wait"] != "42.5" {
		t.Errorf("got %q, want 42.5", out.SetVars["avg_wait"])
	}
}

func TestGetQueueMetrics_StoreError(t *testing.T) {
	store := &fakeQueueStore{historyErr: fmt.Errorf("db error")}
	op := &getQueueMetricsOp{store: store}
	_, err := op.Execute(context.Background(), queueInput(map[string]any{
		"queue":  map[string]any{"id": 1},
		"set":    "val",
		"calls":  "complete",
		"metric": "avg",
	}))
	if err == nil {
		t.Fatal("expected error when store fails")
	}
}

// ── getQueueInfo ──────────────────────────────────────────────────────────────

func TestGetQueueInfo_NoQueue(t *testing.T) {
	op := &getQueueInfoOp{store: &fakeQueueStore{}}
	_, err := op.Execute(context.Background(), queueInput(map[string]any{
		"set": map[string]any{"enabled": "queue_enabled"},
	}))
	if err == nil {
		t.Fatal("expected error when queue is nil")
	}
}

func TestGetQueueInfo_EmptySet(t *testing.T) {
	op := &getQueueInfoOp{store: &fakeQueueStore{}}
	_, err := op.Execute(context.Background(), queueInput(map[string]any{
		"queue": map[string]any{"id": 5},
	}))
	if err == nil {
		t.Fatal("expected error when set is empty")
	}
}

func TestGetQueueInfo_Success(t *testing.T) {
	store := &fakeQueueStore{
		queueData: flow.Variables{"queue_enabled": "true", "queue_priority": "10"},
	}
	op := &getQueueInfoOp{store: store}
	out, err := op.Execute(context.Background(), queueInput(map[string]any{
		"queue": map[string]any{"id": 5},
		"set":   map[string]any{"enabled": "queue_enabled"},
	}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.SetVars["queue_enabled"] != "true" {
		t.Errorf("queue_enabled = %q, want true", out.SetVars["queue_enabled"])
	}
}

func TestGetQueueInfo_StoreError(t *testing.T) {
	store := &fakeQueueStore{queueDataErr: fmt.Errorf("not found")}
	op := &getQueueInfoOp{store: store}
	_, err := op.Execute(context.Background(), queueInput(map[string]any{
		"queue": map[string]any{"id": 5},
		"set":   map[string]any{"x": "y"},
	}))
	if err == nil {
		t.Fatal("expected error when store fails")
	}
}

// ── getQueueAgents ────────────────────────────────────────────────────────────

func TestGetQueueAgents_NoQueue(t *testing.T) {
	op := &getQueueAgentsOp{store: &fakeQueueStore{}}
	_, err := op.Execute(context.Background(), queueInput(map[string]any{
		"set": map[string]any{"x": "y"},
	}))
	if err == nil {
		t.Fatal("expected error when queue.id is nil")
	}
}

func TestGetQueueAgents_Success(t *testing.T) {
	store := &fakeQueueStore{
		agentsData: flow.Variables{"agent_count": "3"},
	}
	op := &getQueueAgentsOp{store: store}
	out, err := op.Execute(context.Background(), queueInput(map[string]any{
		"queue":   map[string]any{"id": 7},
		"channel": "call",
		"set":     map[string]any{"count": "agent_count"},
	}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.SetVars["agent_count"] != "3" {
		t.Errorf("agent_count = %q, want 3", out.SetVars["agent_count"])
	}
}

func TestGetQueueAgents_StoreError(t *testing.T) {
	store := &fakeQueueStore{agentsErr: fmt.Errorf("store error")}
	op := &getQueueAgentsOp{store: store}
	_, err := op.Execute(context.Background(), queueInput(map[string]any{
		"queue": map[string]any{"id": 7},
		"set":   map[string]any{"x": "y"},
	}))
	if err == nil {
		t.Fatal("expected error when store fails")
	}
}
