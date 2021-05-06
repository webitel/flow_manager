package model

import "net/http"

var (
	ErrNotFoundRoute = NewAppError("Route", "app.route.not_found", nil, "not found route", http.StatusNotFound)
)
