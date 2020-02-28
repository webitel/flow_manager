package model

type GRPCConnection interface {
	Id() string
	SchemaId() int
	SchemaUpdatedAt() int64
	DomainId() int
}
