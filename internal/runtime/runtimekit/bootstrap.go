// Package runtimekit provides shared setup helpers for channel routers that
// use the resumable runtime. Each channel (IM, chat, email) calls Bootstrap
// once during Init to get a ready-made Driver + Coordinator pair without
// copy-pasting the registry wiring.
package runtimekit

import (
	"context"

	"github.com/webitel/flow_manager/flow"
	"github.com/webitel/flow_manager/internal/domain/contacts"
	"github.com/webitel/flow_manager/internal/domain/shared/ports"
	"github.com/webitel/flow_manager/internal/runtime/coordinator"
	"github.com/webitel/flow_manager/internal/runtime/interpreter"
	"github.com/webitel/flow_manager/internal/runtime/ops"
	"github.com/webitel/flow_manager/internal/runtime/ops/builtin"
	"github.com/webitel/flow_manager/internal/runtime/ops/domain/calendar"
	casesop "github.com/webitel/flow_manager/internal/runtime/ops/domain/cases"
	contactsop "github.com/webitel/flow_manager/internal/runtime/ops/domain/contacts"
	meetingop "github.com/webitel/flow_manager/internal/runtime/ops/domain/meeting"
	memberop "github.com/webitel/flow_manager/internal/runtime/ops/domain/member"
	notifop "github.com/webitel/flow_manager/internal/runtime/ops/domain/notification"
	schemaop "github.com/webitel/flow_manager/internal/runtime/ops/domain/schema"
	"github.com/webitel/flow_manager/internal/runtime/ops/legacy"
	"github.com/webitel/flow_manager/internal/runtime/tree"
	"github.com/webitel/flow_manager/model"
)

// Config holds the channel-specific inputs that Bootstrap needs to build the
// shared runtime components.
type Config struct {
	// Deps is the channel router's dependency bundle.
	Deps ports.RouterDeps

	// Router is the channel's model.Router implementation — passed to the
	// legacy adapter so existing flow.ApplicationHandlers still work.
	Router model.Router

	// Apps is the merged ApplicationHandlers map for this channel (channel-
	// specific handlers unioned with framework handlers). Bootstrap passes it
	// to legacy.RegisterFromMap; callers must delete any op they override with
	// a native implementation before calling Bootstrap.
	Apps flow.ApplicationHandlers

	// ContactsClient, when non-nil, enables the contacts native ops
	// (getContact, findContact, addContact, updateContact, mergeContactPhones,
	// mergeContactVariables, linkContact). Leave nil to keep legacy bridging.
	ContactsClient contacts.Client

	// ExtraOps, when non-nil, is called with the registry after builtin and
	// legacy ops are registered. Use it to register channel-specific native ops
	// (e.g. messaging.New() for IM).
	ExtraOps func(reg *ops.Registry)

	// LoadTree resolves a schema by (domainID, schemaID) and returns a parsed
	// Tree. Called by the Coordinator on every Dispatch. Callers may add
	// caching here if needed.
	LoadTree func(ctx context.Context, domainID int64, schemaID int) (*tree.Tree, error)
}

// Kit is the result of Bootstrap — a Driver and Coordinator ready to use.
type Kit struct {
	Driver *interpreter.Driver
	Coord  coordinator.Coordinator
}

// Bootstrap builds the registry, driver, and coordinator for a channel router.
// The caller owns the returned Kit and stores it on the router struct.
func Bootstrap(cfg Config) *Kit {
	reg := ops.NewRegistry()
	builtin.Register(reg)
	legacy.RegisterFromMap(reg, cfg.Router, cfg.Apps)

	reg.Register("timezone", builtin.TimezoneOp(cfg.Deps.GetLocation))
	reg.Register("calendar", calendar.New(func(ctx context.Context, domainID int64, id *int, name *string) (*calendar.Result, error) {
		cal, err := cfg.Deps.GetStore().Calendar().Check(domainID, id, name)
		if err != nil {
			return nil, err
		}
		return &calendar.Result{
			Accept:   cal.Accept,
			Expire:   cal.Expire,
			Excepted: cal.Excepted,
		}, nil
	}))

	loadSchemaTr := func(ctx context.Context, domainID int64, schemaID int) (*tree.Tree, error) {
		schema, appErr := cfg.Deps.GetSchemaById(domainID, schemaID)
		if appErr != nil {
			return nil, appErr
		}
		rawSchema := make([]map[string]any, len(schema.Schema))
		for i, app := range schema.Schema {
			rawSchema[i] = map[string]any(app)
		}
		return tree.Parse(schema.Id, rawSchema)
	}
	reg.Register("schema", schemaop.New(loadSchemaTr, reg))
	reg.Register("createMeeting", meetingop.New(cfg.Deps.Meeting()))
	memberop.Register(reg, cfg.Deps.GetStore().Member())
	casesop.Register(reg, cfg.Deps.Cases())
	if cfg.ContactsClient != nil {
		contactsop.Register(reg, cfg.ContactsClient, cfg.Deps)
	}
	reg.Register("notification", notifop.New(cfg.Deps))

	if cfg.ExtraOps != nil {
		cfg.ExtraOps(reg)
	}

	driver := interpreter.NewDriver(
		cfg.Deps.RuntimeStateRepo(),
		reg,
		cfg.Deps.Log(),
		func(ctx context.Context, domainID int64, name string) string {
			return cfg.Deps.SchemaVariable(ctx, domainID, name)
		},
	)

	coord := coordinator.New(cfg.Deps.RuntimeStateRepo(), driver, cfg.LoadTree)

	return &Kit{Driver: driver, Coord: coord}
}
