package model

type Calendar struct {
	Name     string  `json:"name" db:"name"`
	Excepted *string `json:"excepted" db:"excepted"`
	Accept   bool    `json:"accept" db:"accept"`
	Expire   bool    `json:"expire"`
}
