package legacy

import (
	"github.com/webitel/flow_manager/flow"
	"github.com/webitel/flow_manager/internal/runtime/ops"
)

// nativeOps lists op names handled by the builtin interpreter; never bridge these.
var nativeOps = map[string]bool{
	"if":     true,
	"while":  true,
	"switch": true,
	"goto":   true,
	"break":  true,
	"set":    true,
	"log":    true,
	"start":  true,
}

// RegisterLegacy wraps every handler from router as a LegacyOp in reg,
// skipping any name already covered by native builtins.
func RegisterLegacy(reg *ops.Registry, router flow.Router) {
	for name, app := range router.Handlers() {
		if nativeOps[name] {
			continue
		}
		app := app // capture
		reg.Register(name, &LegacyOp{name: name, app: app, router: router})
	}
}
