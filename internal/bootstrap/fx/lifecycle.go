package bsfx

import (
	"context"

	"go.uber.org/fx"

	bsruntime "github.com/webitel/flow_manager/internal/bootstrap/runtime"
)

// RegisterStartupHooks wires FlowManager.Start into fx.Lifecycle so that all
// I/O-bound startup steps run inside OnStart rather than during construction.
func RegisterStartupHooks(lc fx.Lifecycle, fm *bsruntime.FlowManager) {
	lc.Append(fx.Hook{
		OnStart: func(_ context.Context) error {
			return fm.Start()
		},
	})
}
