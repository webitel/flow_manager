package model

import "github.com/webitel/flow_manager/internal/bootstrap/config"

// Re-exports for backward compatibility.
var AppServiceName = config.AppServiceName

const (
	HeaderFromServiceName    = config.HeaderFromServiceName
	SchemaCacheSize          = config.SchemaCacheSize
	SchemaCacheExpire        = config.SchemaCacheExpire
	AppServiceTTL            = config.AppServiceTTL
	AppDeregisterCriticalTTL = config.AppDeregisterCriticalTTL
)
