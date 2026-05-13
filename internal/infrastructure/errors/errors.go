package errors

import (
	"errors"
	"fmt"
	"net/http"
)

// StatusError carries an HTTP status code alongside an error message.
// Use it only where the caller needs to distinguish status semantically
// (e.g. 400 vs 404 vs 500). For internal errors use fmt.Errorf.
type StatusError struct {
	Code int
	Err  error
}

func (e *StatusError) Error() string { return e.Err.Error() }
func (e *StatusError) Unwrap() error { return e.Err }

// New creates a StatusError with the given HTTP code and message.
func New(code int, msg string) error {
	return &StatusError{Code: code, Err: errors.New(msg)}
}

// Newf creates a StatusError with a formatted message.
func Newf(code int, format string, args ...any) error {
	return &StatusError{Code: code, Err: fmt.Errorf(format, args...)}
}

// Wrap wraps an existing error with an HTTP status code.
func Wrap(code int, err error) error {
	if err == nil {
		return nil
	}
	return &StatusError{Code: code, Err: err}
}

// CodeOf returns the HTTP status code from a StatusError, or 500 if not found.
func CodeOf(err error) int {
	var se *StatusError
	if errors.As(err, &se) {
		return se.Code
	}
	return http.StatusInternalServerError
}
