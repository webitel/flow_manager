package bsfx

import (
	"context"

	"go.uber.org/fx"

	"github.com/webitel/flow_manager/app"
)

// RegisterStartupHooks wires FlowManager.Start into fx.Lifecycle so that all
// I/O-bound startup steps run inside OnStart rather than during construction.
func RegisterStartupHooks(lc fx.Lifecycle, fm *app.FlowManager) {
	lc.Append(fx.Hook{
		OnStart: func(_ context.Context) error {
			return fm.Start()
		},
	})
}
