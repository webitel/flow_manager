package coordinator_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/webitel/flow_manager/internal/runtime/coordinator"
	"github.com/webitel/flow_manager/internal/runtime/persistence"
	"github.com/webitel/flow_manager/internal/runtime/state"
	"github.com/webitel/flow_manager/internal/runtime/tree"
)

// --- fakes ---

type fakeRepo struct {
	byKey map[string]*persistence.Record
	err   error
}

func (f *fakeRepo) LoadByResumeKey(_ context.Context, key string) (*persistence.Record, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.byKey[key], nil
}

type fakeDriver struct {
	calls     []resumeCall
	resumeErr error
}

type resumeCall struct {
	rec     *persistence.Record
	payload map[string]string
}

func (f *fakeDriver) Resume(_ context.Context, rec *persistence.Record, _ *tree.Tree, payload map[string]string) error {
	f.calls = append(f.calls, resumeCall{rec: rec, payload: payload})
	return f.resumeErr
}

func minimalTree() *tree.Tree {
	tr, _ := tree.Parse(1, tree.Schema{{"set": map[string]any{"x": "1"}}})
	return tr
}

func suspendedRecord(key string) *persistence.Record {
	return &persistence.Record{
		ID:        uuid.New(),
		ResumeKey: key,
		DomainID:  1,
		SchemaID:  1,
		Status:    state.StatusSuspended,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

// --- tests ---

func TestDispatch_ResumesRecord(t *testing.T) {
	rec := suspendedRecord("msg:conn-1")
	repo := &fakeRepo{byKey: map[string]*persistence.Record{"msg:conn-1": rec}}
	driver := &fakeDriver{}
	loader := func(_ context.Context, _ int64, _ int) (*tree.Tree, error) { return minimalTree(), nil }

	c := coordinator.New(repo, driver, loader)

	payload := map[string]string{"msg": "hello", "from": "user-42"}
	if err := c.Dispatch(context.Background(), "msg:conn-1", payload); err != nil {
		t.Fatalf("Dispatch: %v", err)
	}

	if len(driver.calls) != 1 {
		t.Fatalf("expected 1 Resume call, got %d", len(driver.calls))
	}
	if driver.calls[0].rec != rec {
		t.Error("Resume was called with wrong record")
	}
	if driver.calls[0].payload["msg"] != "hello" {
		t.Errorf("wrong payload: %v", driver.calls[0].payload)
	}
}

func TestDispatch_StaleKey_NoResume(t *testing.T) {
	repo := &fakeRepo{byKey: map[string]*persistence.Record{}}
	driver := &fakeDriver{}
	loader := func(_ context.Context, _ int64, _ int) (*tree.Tree, error) {
		t.Error("loadTree must not be called for stale key")
		return nil, nil
	}

	c := coordinator.New(repo, driver, loader)

	if err := c.Dispatch(context.Background(), "msg:unknown", nil); err != nil {
		t.Fatalf("Dispatch on stale key should return nil, got: %v", err)
	}
	if len(driver.calls) != 0 {
		t.Error("Resume must not be called for stale key")
	}
}

func TestDispatch_RepoError_PropagatesError(t *testing.T) {
	want := errors.New("db is down")
	repo := &fakeRepo{err: want}
	driver := &fakeDriver{}
	loader := func(_ context.Context, _ int64, _ int) (*tree.Tree, error) { return nil, nil }

	c := coordinator.New(repo, driver, loader)

	err := c.Dispatch(context.Background(), "any:key", nil)
	if err == nil {
		t.Fatal("expected error from repo, got nil")
	}
	if !errors.Is(err, want) {
		t.Errorf("expected %v wrapped in error, got %v", want, err)
	}
}

func TestDispatch_NilPayload_Allowed(t *testing.T) {
	rec := suspendedRecord("timer:abc")
	repo := &fakeRepo{byKey: map[string]*persistence.Record{"timer:abc": rec}}
	driver := &fakeDriver{}
	loader := func(_ context.Context, _ int64, _ int) (*tree.Tree, error) { return minimalTree(), nil }

	c := coordinator.New(repo, driver, loader)

	if err := c.Dispatch(context.Background(), "timer:abc", nil); err != nil {
		t.Fatalf("Dispatch with nil payload: %v", err)
	}
	if len(driver.calls) != 1 {
		t.Fatalf("expected 1 Resume call")
	}
	if driver.calls[0].payload != nil {
		t.Errorf("expected nil payload forwarded, got %v", driver.calls[0].payload)
	}
}
