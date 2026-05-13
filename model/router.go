package model

import "github.com/webitel/flow_manager/internal/domain/flow"

// Re-export for backward compatibility.
type ApplicationRequest = flow.ApplicationRequest

// Router dispatches a Connection through a domain-specific flow.
type Router interface {
	Handle(conn Connection) error
	GlobalVariable(domainId int64, name string) string
}
