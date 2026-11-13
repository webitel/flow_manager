package sql

import (
	"errors"
	"fmt"
)

var (
	ErrParameterRequired = errors.New("parameter required")
	ErrOnlyGlobalVars    = errors.New("only global variables ($${...}) are allowed")
	ErrUnsupportedDriver = errors.New("unsupported driver")
	ErrQueryContainsVars = errors.New("vars in query are forbidden")
)

type Error struct {
	err   error
	param string
}

func newRequiredError(param string) *Error {
	return &Error{
		err:   ErrParameterRequired,
		param: param,
	}
}

func (e *Error) Unwrap() error { return e.err }

func (e *Error) Error() string {
	if e.param == "" {
		return fmt.Sprintf("sql: %v", e.err.Error())
	}

	return fmt.Sprintf("sql: %q %v", e.param, e.err.Error())
}
