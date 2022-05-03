package model

type AttemptResult struct {
	Id            int64
	Status        string
	Description   string
	ReadyAt       *int64
	ExpiredAt     *int64
	Variables     map[string]string
	StickyDisplay bool
	AgentId       int32
}
