package model

import "github.com/webitel/flow_manager/internal/domain/flow"

// Re-export for backward compatibility.
type ApplicationRequest = flow.ApplicationRequest

// Router dispatches a Connection through a domain-specific flow.
// Kept here (not aliased) because Handle() returns *AppError — moving it would
// create an import cycle until AppError is extracted (Phase 5.2).
type Router interface {
	Handle(conn Connection) *AppError
	GlobalVariable(domainId int64, name string) string
}
