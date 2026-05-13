package routing

// moved from model/schema.go — see model/schema.go for re-export aliases

import (
	"fmt"

	"github.com/webitel/flow_manager/internal/domain/flow"
)

const (
	FlowSchemaNameVariable = "flow_schema_name"
)

// Schema holds a compiled flow schema ready for execution.
type Schema struct {
	Id        int   `json:"id" db:"id"`
	DomainId  int64 `json:"domain_id" db:"domain_id"`
	UpdatedAt int64 `json:"updated_at" db:"updated_at"`

	Type       string            `json:"type" db:"type"`
	DomainName string            `json:"domain_name" db:"domain_name"`
	Name       string            `json:"name" db:"name"`
	Schema     flow.Applications `json:"schema" db:"schema"`
	Debug      bool              `json:"debug" db:"debug"`
}

// SchemaVariable holds a potentially encrypted schema variable.
type SchemaVariable struct {
	Encrypt       bool   `json:"encrypt" db:"encrypt"`
	Value         []byte `json:"value" db:"value"`
	ComputedValue string `json:"computed_value" db:"-"`
}

func (s *Schema) Hash() string {
	return fmt.Sprintf("%d.%d", s.Id, s.DomainId)
}
