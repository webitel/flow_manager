package model

type GRPCConnection interface {
	Id() string
	DomainId() int
}
