package processing

import (
	procop "github.com/webitel/flow_manager/internal/runtime/ops/domain/processing"
	"github.com/webitel/flow_manager/internal/runtime/runtimekit"
)

// Deps is the narrow interface that the processing router and its ops need.
// *bsruntime.RouterDeps satisfies this interface.
type Deps interface {
	runtimekit.BootstrapDeps
	AppID() string
	procop.AttemptDeps
}
