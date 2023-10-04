package model

type User struct {
	Name      *string           `json:"name" db:"name"`
	Dnd       *bool             `json:"dnd" db:"dnd"`
	Extension *string           `json:"extension" db:"extension"`
	Variables map[string]string `json:"variables" db:"variables"`
}

type SearchUser struct {
	Id        *int    `json:"id"`
	Name      *string `json:"name"`
	Extension *string `json:"extension" db:"extension"`
}
