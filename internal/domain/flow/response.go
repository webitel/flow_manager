package flow

// moved from model/applications.go — see model/applications.go for re-export aliases

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
)

// ApplicationObject is a single application configuration map.
type ApplicationObject map[string]interface{}

// Applications is an ordered list of application configurations.
type Applications []ApplicationObject

// Response is the result returned by an application handler.
type Response interface {
	String() string
}

func (j Applications) Value() (driver.Value, error) {
	str, err := json.Marshal(j)
	return string(str), err
}

func (j *Applications) Scan(src interface{}) error {
	if bytes, ok := src.([]byte); ok {
		return json.Unmarshal(bytes, &j)
	}
	return errors.New(fmt.Sprintf("unmarshal %v", src))
}
