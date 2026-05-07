package runtimekit

import (
	"context"
	"fmt"

	"github.com/webitel/flow_manager/internal/runtime/persistence"
	"github.com/webitel/flow_manager/internal/runtime/sessionmgr"
	"github.com/webitel/flow_manager/internal/runtime/state"
	"github.com/webitel/flow_manager/internal/runtime/tree"
	"github.com/webitel/flow_manager/model"
	"github.com/webitel/wlog"
)

// FlowRunner executes a flow for a given runtime state record.
// *interpreter.Driver satisfies this interface.
type FlowRunner interface {
	Run(ctx context.Context, rec *persistence.Record, tr *tree.Tree, payload map[string]string) error
}

// SessionWatcher registers a suspended connection with the session manager.
// *sessionmgr.Manager satisfies this interface.
type SessionWatcher interface {
	Watch(
		conn sessionmgr.Connection,
		rec *persistence.Record,
		initialMsg string,
		decorator sessionmgr.ContextDecorator,
		teardown sessionmgr.TeardownFunc,
	)
}

// HandleConfig groups the channel-specific inputs for RunSession.
type HandleConfig struct {
	// ChannelName is used in log messages (e.g. "im", "chat").
	ChannelName string
	// ChannelType is stored in persistence.Record.Channel.
	ChannelType int16
	// Conn is the original model.Connection.
	Conn model.Connection
	// Tr is the parsed schema tree.
	Tr *tree.Tree
	// Tags maps tag name → node ID (from Tr.ByTag).
	Tags map[string]string
	// SchemaID, DomainID, AppID are used when creating a fresh record.
	SchemaID int
	DomainID int64
	AppID    string
	// Repo is used to load and create runtime_state records.
	Repo persistence.Repository
	// Driver executes the flow.
	Driver FlowRunner
	// SessionMgr watches suspended sessions.
	SessionMgr SessionWatcher
	// Decorator injects channel-specific values into the dispatch context.
	// Also applied to the runCtx passed to Driver.Run. May be nil.
	Decorator sessionmgr.ContextDecorator
	// Teardown is called exactly once when the session ends.
	Teardown sessionmgr.TeardownFunc
	// Log is required for warnings.
	Log *wlog.Logger
}

// RunSession drives the recovery / fresh-start / suspend-watch state machine
// for a resumable channel session.
//
// Call this after: feature-flag check, checkpoint creation, and Teardown
// closure are all set up by the channel router.
//
// Returns (true, nil) when sessionmgr is watching — Teardown will fire
// asynchronously; the caller must NOT call it.
//
// Returns (false, err) when the persistence.Record could not be created. Teardown
// was NOT called. The caller must stop the connection with a channel-specific
// error and may return without further cleanup.
//
// Returns (false, nil) when the flow completed synchronously — Teardown was
// called inside RunSession.
func RunSession(rec *persistence.Record, cfg HandleConfig) (watching bool, err error) {
	conn := cfg.Conn
	ctx := conn.Context()

	decorate := cfg.Decorator
	if decorate == nil {
		decorate = func(c context.Context) context.Context { return c }
	}

	sessConn, ok := conn.(sessionmgr.Connection)
	if !ok {
		cfg.Log.Warn(fmt.Sprintf("%s handle: connection %s does not satisfy sessionmgr.Connection",
			cfg.ChannelName, conn.Id()))
		cfg.Teardown()
		return false, nil
	}

	// Recovery: reconnected to a suspended flow — skip Run entirely.
	if rec != nil && rec.Status == state.StatusSuspended {
		initialMsg := conn.Variables()[model.ConversationStartMessageVariable]
		cfg.SessionMgr.Watch(sessConn, rec, initialMsg, decorate, cfg.Teardown)
		return true, nil
	}

	// Fresh start: seed state from connection variables and persist.
	if rec == nil {
		es := state.NewExecState(cfg.SchemaID, cfg.Tr.Version, cfg.Tags)
		for k, v := range conn.Variables() {
			es.Variables[k] = v
		}
		rec = &persistence.Record{
			ConnectionID: conn.Id(),
			DomainID:     cfg.DomainID,
			Channel:      cfg.ChannelType,
			SchemaID:     cfg.SchemaID,
			AppID:        cfg.AppID,
			State:        es,
			Status:       state.StatusRunning,
		}
		if createErr := cfg.Repo.Create(ctx, rec); createErr != nil {
			return false, createErr
		}
	}

	// Run the driver. Decorator provides the same enriched context used for
	// Watch dispatches (legacy connection ref, connID, etc.).
	runCtx := decorate(ctx)
	if runErr := cfg.Driver.Run(runCtx, rec, cfg.Tr, nil); runErr != nil {
		cfg.Log.Error(fmt.Sprintf("%s driver.Run conn=%s: %v", cfg.ChannelName, conn.Id(), runErr))
	}

	// Flow suspended mid-run — hand off to sessionmgr.
	if rec.Status == state.StatusSuspended {
		cfg.SessionMgr.Watch(sessConn, rec, "", decorate, cfg.Teardown)
		return true, nil
	}

	cfg.Teardown()
	return false, nil
}
