package model

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
)

type Routing struct {
	SourceId        int    `json:"source_id" db:"source_id"`
	SourceName      string `json:"source_name" db:"source_name"`
	SourceData      string `json:"source_data" db:"source_data"`
	DomainId        int64  `json:"domain_id" db:"domain_id"`
	DomainName      string `json:"domain_name" db:"domain_name"`
	Number          string `json:"number" db:"number"`
	TimezoneId      int64  `json:"timezone_id" db:"timezone_id"`
	TimezoneName    string `json:"timezone_name" db:"timezone_name"`
	SchemaId        int    `json:"scheme_id" db:"scheme_id"`
	SchemaName      string `json:"scheme_name" db:"scheme_name"`
	SchemaUpdatedAt int64  `json:"schema_updated_at" db:"schema_updated_at"`
	Schema          *Schema

	Variables *vars `json:"variables" db:"variables"`
	Debug     bool  `json:"debug" db:"debug"`
}

type vars map[string]string

func (j vars) Value() (driver.Value, error) {
	str, err := json.Marshal(j)
	return string(str), err
}

func (j *vars) Scan(src interface{}) error {
	if bytes, ok := src.([]byte); ok {
		return json.Unmarshal(bytes, &j)
	}
	return errors.New("Error")
}
