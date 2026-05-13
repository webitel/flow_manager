package model

import "github.com/webitel/flow_manager/internal/domain/flow"

// Re-exports for backward compatibility.
type Variables = flow.Variables
type ParseOption = flow.ParseOption

const ParseOptionJson = flow.ParseOptionJson

func ParseText(c Connection, text string, ops ...ParseOption) string {
	return flow.ParseText(c, text, ops...)
}
