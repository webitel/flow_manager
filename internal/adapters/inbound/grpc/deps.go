package grpc

import (
	"github.com/webitel/flow_manager/internal/runtime/runtimekit"
	"github.com/webitel/flow_manager/model"
)

// Deps is the narrow interface that the gRPC router needs.
// *app.FlowManager satisfies this interface.
type Deps interface {
	runtimekit.BootstrapDeps
	AppID() string
	StoreCallVariables(id string, vars map[string]string) *model.AppError
}
