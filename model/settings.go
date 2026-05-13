package model

import "github.com/webitel/flow_manager/internal/bootstrap/config"

// Re-exports for backward compatibility.
type SysValue = config.SysValue

const (
	SysAutoLinkCallToContact = config.SysAutoLinkCallToContact
	SysAutoLinkMailToContact = config.SysAutoLinkMailToContact
)
