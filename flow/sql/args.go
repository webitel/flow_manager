package sql

import (
	"strings"
	"time"
)

const defaultExternalSQLTimeout int = 1000

type Args struct {
	Driver  string `json:"driver"`
	DSN     string `json:"dns"`
	Query   string `json:"query"`
	Params  []any  `json:"params"`
	Timeout int    `json:"timeout"`
}

func DefaultSqlArgs() *Args { return &Args{Timeout: defaultExternalSQLTimeout} }

func (a *Args) QueryTimeout() time.Duration {
	return time.Duration(a.Timeout) * time.Millisecond
}

func (a *Args) Normalize() {
	if a == nil {
		return
	}

	a.Driver = strings.TrimSpace(strings.ToLower(a.Driver))

	if a.Timeout < 0 {
		a.Timeout = defaultExternalSQLTimeout
	}
}

func (a *Args) Validate() error {
	if a == nil {
		return newRequiredError("args")
	}

	if strings.TrimSpace(a.Query) == "" {
		return newRequiredError("query")
	}

	if a.Driver == "" {
		return newRequiredError("driver")
	}

	if _, supports := supportedDrivers[a.Driver]; !supports {
		return &Error{err: ErrUnsupportedDriver, param: a.Driver}
	}

	if strings.TrimSpace(a.DSN) == "" {
		return newRequiredError("dsn")
	}

	return nil
}

func (a *Args) Prepare(lookup globalVarLookup) error {
	if compileVars.MatchString(a.Query) {
		return &Error{err: ErrQueryContainsVars}
	}

	driver, err := resolveGlobalVars(a.Driver, lookup)
	if err != nil {
		return &Error{err: err, param: "driver"}
	}

	a.Driver = driver

	dsn, err := resolveGlobalVars(a.DSN, lookup)
	if err != nil {
		return &Error{err: err, param: "dsn"}
	}

	a.DSN = dsn

	return nil
}
