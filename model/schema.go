package model

import "fmt"

type Schema struct {
	Id        int   `json:"id" db:"id"`
	DomainId  int   `json:"domain_id" db:"domain_id"`
	UpdatedAt int64 `json:"updated_at" db:"updated_at"`

	Type       int          `json:"type" db:"type"`
	DomainName string       `json:"domain_name" db:"domain_name"`
	Name       string       `json:"name" db:"name"`
	Schema     Applications `json:"schema" db:"schema"`
	Debug      bool         `json:"debug" db:"debug"`
}

func (s *Schema) Hash() string {
	return fmt.Sprintf("%d.%d", s.Id, s.DomainId)
}
