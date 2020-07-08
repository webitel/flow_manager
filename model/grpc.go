package model

type GRPCConnection interface {
	Connection
	SchemaId() int
	Result(result interface{})
}
