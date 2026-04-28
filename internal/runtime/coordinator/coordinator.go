// Package coordinator provides the single re-entry point for external events
// (inbound messages, queue events, timer expiry) that resume a suspended flow.
//
// All event sources call Coordinator.Dispatch; the coordinator locates the
// suspended record, loads the schema tree, and delegates to the Driver.
package coordinator

import (
	"context"
	"fmt"

	"github.com/webitel/flow_manager/internal/runtime/persistence"
	"github.com/webitel/flow_manager/internal/runtime/tree"
)

// ResumeKeyLoader is the minimal repository subset the Coordinator needs.
type ResumeKeyLoader interface {
	LoadByResumeKey(ctx context.Context, key string) (*persistence.Record, error)
}

// Resumable is satisfied by *interpreter.Driver. Defined here to avoid a
// circular import and to allow lightweight fakes in tests.
type Resumable interface {
	Resume(ctx context.Context, rec *persistence.Record, tr *tree.Tree, payload map[string]string) error
}

// SchemaLoader fetches and parses the execution tree for a schema. Injected
// per-channel at construction time; the coordinator does not care which
// channel is involved.
type SchemaLoader func(ctx context.Context, domainID int64, schemaID int) (*tree.Tree, error)

// Coordinator is the public contract.
type Coordinator interface {
	// Dispatch resumes the suspended flow identified by resumeKey, passing
	// payload as OpInput.ResumePayload to the first op that executes.
	// Returns nil when no suspended record is found (stale / already resumed).
	Dispatch(ctx context.Context, resumeKey string, payload map[string]string) error
}

type coordinator struct {
	repo     ResumeKeyLoader
	driver   Resumable
	loadTree SchemaLoader
}

// New constructs a Coordinator. All three arguments are required.
func New(repo ResumeKeyLoader, driver Resumable, loadTree SchemaLoader) Coordinator {
	return &coordinator{repo: repo, driver: driver, loadTree: loadTree}
}

func (c *coordinator) Dispatch(ctx context.Context, resumeKey string, payload map[string]string) error {
	rec, err := c.repo.LoadByResumeKey(ctx, resumeKey)
	if err != nil {
		return fmt.Errorf("coordinator: load by resume key %q: %w", resumeKey, err)
	}
	if rec == nil {
		// Stale or already-resumed key — safe to drop.
		return nil
	}

	tr, err := c.loadTree(ctx, rec.DomainID, rec.SchemaID)
	if err != nil {
		return fmt.Errorf("coordinator: load tree schema=%d domain=%d: %w", rec.SchemaID, rec.DomainID, err)
	}

	return c.driver.Resume(ctx, rec, tr, payload)
}
