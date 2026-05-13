// Package runtimekit provides shared setup helpers for channel routers that
// use the resumable runtime. Each channel (IM, chat, email) calls Bootstrap
// once during Init to get a ready-made Driver + Coordinator pair without
// copy-pasting the registry wiring.
package runtimekit

import (
	"context"
	"time"

	"github.com/webitel/wlog"

	"github.com/webitel/flow_manager/internal/domain/contacts"
	domcases "github.com/webitel/flow_manager/internal/domain/cases"
	domainmeeting "github.com/webitel/flow_manager/internal/domain/meeting"
	"github.com/webitel/flow_manager/internal/runtime/coordinator"
	"github.com/webitel/flow_manager/internal/runtime/interpreter"
	"github.com/webitel/flow_manager/internal/runtime/ops"
	"github.com/webitel/flow_manager/internal/runtime/ops/builtin"
	"github.com/webitel/flow_manager/internal/runtime/ops/domain/calendar"
	casesop "github.com/webitel/flow_manager/internal/runtime/ops/domain/cases"
	contactsop "github.com/webitel/flow_manager/internal/runtime/ops/domain/contacts"
	emailop "github.com/webitel/flow_manager/internal/runtime/ops/domain/email"
	meetingop "github.com/webitel/flow_manager/internal/runtime/ops/domain/meeting"
	memberop "github.com/webitel/flow_manager/internal/runtime/ops/domain/member"
	notifop "github.com/webitel/flow_manager/internal/runtime/ops/domain/notification"
	queueop "github.com/webitel/flow_manager/internal/runtime/ops/domain/queue"
	schemaop "github.com/webitel/flow_manager/internal/runtime/ops/domain/schema"
	"github.com/webitel/flow_manager/internal/runtime/persistence"
	"github.com/webitel/flow_manager/internal/runtime/tree"
	"github.com/webitel/flow_manager/model"
	"github.com/webitel/flow_manager/store"
)

// BootstrapDeps is the set of dependencies that Bootstrap consumes directly
// (own calls + interfaces required by the ops it registers).
type BootstrapDeps interface {
	// direct calls
	GetLocation(id int) *time.Location
	GetStore() store.Store
	GetSchemaById(domainId int64, id int) (*model.Schema, error)
	Meeting() domainmeeting.Client
	Cases() domcases.Client
	RuntimeStateRepo() persistence.Repository
	Log() *wlog.Logger
	SchemaVariable(ctx context.Context, domainID int64, name string) string

	// op interfaces (ops are passed cfg.Deps directly)
	builtin.CookieCache
	builtin.GlobalDeps
	builtin.ListDeps
	builtin.CacheDeps
	builtin.GenerateLinkDeps
	builtin.OpenLinkDeps
	builtin.SqlDeps
	emailop.EmailDeps
	contactsop.LinkDeps
	notifop.Deps
}

// Config holds the channel-specific inputs that Bootstrap needs to build the
// shared runtime components.
type Config struct {
	// Deps is the channel router's dependency bundle.
	Deps BootstrapDeps

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

	reg.Register("httpRequest", builtin.HTTPRequestOp(cfg.Deps))
	reg.Register("global", builtin.GlobalOp(cfg.Deps))
	reg.Register("list", builtin.ListOp(cfg.Deps))
	reg.Register("listAdd", builtin.ListAddOp(cfg.Deps))
	reg.Register("cache", builtin.CacheOp(cfg.Deps))
	reg.Register("generateLink", builtin.GenerateLinkOp(cfg.Deps))
	reg.Register("openLink", builtin.OpenLinkOp(cfg.Deps))
	reg.Register("sql", builtin.SqlOp(cfg.Deps))
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
	queueop.Register(reg, cfg.Deps.GetStore().Queue())
	emailop.Register(reg, cfg.Deps)
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
