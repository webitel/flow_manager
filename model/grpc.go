package model

import "context"

type GRPCConnection interface {
	Connection
	SchemaId() int
	Result(result interface{})
	Export(ctx context.Context, vars []string) (Response, *AppError)
	DumpExportVariables() map[string]string
	Scope() Scope
}
