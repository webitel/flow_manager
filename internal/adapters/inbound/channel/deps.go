package channel

import (
	"github.com/webitel/flow_manager/internal/runtime/runtimekit"
)

// Deps is the narrow interface that the channel router needs.
// *bsruntime.RouterDeps satisfies this interface.
type Deps interface {
	runtimekit.BootstrapDeps
	AppID() string
}
