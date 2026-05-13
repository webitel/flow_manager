package flow

// moved from model/variables.go — see model/variables.go for re-export aliases

// Variables is a generic key-value map used as flow scope variables.
type Variables map[string]interface{}

// ParseOption controls optional behaviour of ParseText.
type ParseOption uint

const (
	ParseOptionJson ParseOption = 1 << iota
)
