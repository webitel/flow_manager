package model

import (
	"fmt"
	"github.com/lib/pq"
	"strings"
)

type Endpoint struct {
	Id          *int           `json:"id" db:"id"`
	Name        *string        `json:"name" db:"name"`
	Idx         int            `json:"idx" db:"idx"`
	TypeName    string         `json:"type_name" db:"type_name"`
	Dnd         *bool          `json:"dnd" db:"dnd"`
	Destination *string        `json:"destination" db:"destination"`
	Number      *string        `json:"number" db:"-"`
	Variables   pq.StringArray `json:"variables" db:"variables"`
}

func (e *Endpoint) ToStringVariables() string {
	vars := make([]string, len(e.Variables), len(e.Variables)+3)
	vars = e.Variables
	if e.Id != nil {
		vars = append(vars, fmt.Sprintf("wbt_to_id=%d", *e.Id))
	}

	if e.Name != nil {
		vars = append(vars, fmt.Sprintf("wbt_to_name='%s'", *e.Name))
	}

	if e.Number != nil {
		vars = append(vars, fmt.Sprintf("wbt_to_number='%s'", *e.Number))
	} else if e.Destination != nil {
		vars = append(vars, fmt.Sprintf("wbt_to_number='%s'", *e.Destination))
	}

	return fmt.Sprintf("wbt_to_type=%s,%s", e.TypeName, strings.Join(vars, ","))
}
