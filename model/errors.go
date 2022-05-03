package model

import (
	"fmt"
	"net/http"
)

var (
	ErrNotFoundRoute = NewAppError("Route", "app.route.not_found", nil, "not found route", http.StatusNotFound)
)

func ErrorRequiredParameter(appId string, param string) *AppError {
	return NewAppError("Valid", "valid.app."+appId, nil, fmt.Sprintf("App=%s %s is required", appId, param), http.StatusBadRequest)
}
