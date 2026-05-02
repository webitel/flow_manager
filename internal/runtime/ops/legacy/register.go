package legacy

import (
	"github.com/webitel/flow_manager/flow"
	"github.com/webitel/flow_manager/internal/runtime/ops"
	"github.com/webitel/flow_manager/model"
)

// nativeOps lists op names handled by the builtin interpreter; never bridge these.
var nativeOps = map[string]bool{
	"if":       true,
	"while":    true,
	"switch":   true,
	"goto":     true,
	"break":    true,
	"set":      true,
	"log":      true,
	"start":    true,
	"string":   true,
	"math":     true,
	"schema":   true,
	"calendar": true,
}

// RegisterLegacy wraps every handler from router as a LegacyOp in reg,
// skipping any name already covered by native builtins.
func RegisterLegacy(reg *ops.Registry, router flow.Router) {
	RegisterFromMap(reg, router, router.Handlers())
}

// RegisterFromMap is like RegisterLegacy but accepts the apps map separately.
// Use this when the router is available only as model.Router (not flow.Router),
// or when registering a hand-picked subset of handlers.
//
// Example — register only call's echo op:
//
//	apps := call.ApplicationsHandlers(callRouter)
//	legacy.RegisterFromMap(reg, callRouter, flow.ApplicationHandlers{"echo": apps["echo"]})
func RegisterFromMap(reg *ops.Registry, router model.Router, apps flow.ApplicationHandlers) {
	for name, app := range apps {
		if nativeOps[name] {
			continue
		}
		app := app // capture
		reg.Register(name, &LegacyOp{name: name, app: app, router: router})
	}
}
