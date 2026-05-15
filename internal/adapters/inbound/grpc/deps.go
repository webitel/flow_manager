package grpc

import (
	"github.com/webitel/flow_manager/internal/runtime/runtimekit"
)

// Deps is the narrow interface that the gRPC router needs.
// *bsruntime.RouterDeps satisfies this interface.
type Deps interface {
	runtimekit.BootstrapDeps
	AppID() string
	StoreCallVariables(id string, vars map[string]string) error
}
