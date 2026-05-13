package user

// moved from model/user.go — see model/user.go for re-export aliases

// User represents a platform user with basic attributes.
type User struct {
	Name      *string           `json:"name" db:"name"`
	Dnd       *bool             `json:"dnd" db:"dnd"`
	Extension *string           `json:"extension" db:"extension"`
	Variables map[string]string `json:"variables" db:"variables"`
}

// SearchUser is a filter for searching users.
type SearchUser struct {
	Id        *int    `json:"id"`
	Name      *string `json:"name"`
	Extension *string `json:"extension" db:"extension"`
	AgentId   *int    `json:"agentId"`
}
