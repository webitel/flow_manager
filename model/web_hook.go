package model

import (
	"net/http"

	apperrs "github.com/webitel/flow_manager/internal/infrastructure/errors"
	webhookdomain "github.com/webitel/flow_manager/internal/domain/webhook"
)

// Re-exports for backward compatibility.
type WebHook = webhookdomain.WebHook
type JsonValue = webhookdomain.JsonValue

const hookAuthHeader = "Authorization"

// WebHookAuthentication checks the Authorization header of r against the hook's
// configured secret.
func WebHookAuthentication(w *WebHook, r *http.Request) error {
	if w.Authorization == nil {
		return nil
	}

	res := r.Header.Get(hookAuthHeader)
	if res != *w.Authorization {
		return apperrs.New(http.StatusUnauthorized, "WebHook: hook.authentication.unauthorized: Unauthorized")
	}

	return nil
}
