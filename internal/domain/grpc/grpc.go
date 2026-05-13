package grpc

import (
	"context"

	"github.com/webitel/flow_manager/internal/domain/flow"
)

// GRPCConnection is a flow Connection specialised for gRPC-driven channels.
type GRPCConnection interface {
	flow.Connection
	SchemaId() int
	Result(result interface{})
	Export(ctx context.Context, vars []string) (flow.Response, error)
	DumpExportVariables() map[string]string
	Scope() flow.Scope
}
