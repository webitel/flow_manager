package session

import (
	"context"
	"time"

	"github.com/webitel/wlog"

	"github.com/webitel/flow_manager/internal/domain/flow"
)

// Save persists a new checkpoint for stateful connections.
// Returns nil if the connection type is not stateful or the repo is not configured.
func Save(repo Repository, appID string, conn flow.Connection, schemaID int) *Checkpoint {
	if !IsStateful(conn.Type()) || repo == nil {
		return nil
	}

	cp := New(conn, schemaID, appID)

	if err := repo.Save(conn.Context(), cp); err != nil {
		conn.Log().Warn("save session checkpoint", wlog.Err(err))
		return nil
	}

	return cp
}

// Update refreshes checkpoint variables after a flow step completes.
func Update(repo Repository, cp *Checkpoint, conn flow.Connection) {
	if cp == nil {
		return
	}

	cp.Refresh(conn.Variables())

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := repo.Update(ctx, cp); err != nil {
		conn.Log().Warn("update session checkpoint", wlog.Err(err))
	}
}

// Close marks the checkpoint as closed after a flow finishes.
func Close(repo Repository, log *wlog.Logger, cp *Checkpoint, connectionID string) {
	if cp == nil {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := repo.Close(ctx, connectionID); err != nil {
		log.Warn("close session checkpoint", wlog.String("connection_id", connectionID), wlog.Err(err))
	}
}
