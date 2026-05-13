package flow

// moved from model/scope.go — see model/scope.go for re-export alias

// Scope identifies the channel and session for a running flow.
type Scope struct {
	Channel string
	Id      string
}
