package routing

import "errors"

// ErrNotFoundRoute is returned when no routing record matches the connection.
// Callers use errors.Is to detect it.
var ErrNotFoundRoute = errors.New("not found route")

// ErrorRequiredParameter returns a 400 Bad Request error for a missing parameter.
func ErrorRequiredParameter(appId string, param string) error {
	return errors.New("valid.app." + appId + ": App=" + appId + " " + param + " is required")
}
