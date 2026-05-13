package model

import (
	"net/http"

	webhookdomain "github.com/webitel/flow_manager/internal/domain/webhook"
)

// Re-exports for backward compatibility.
type WebHook = webhookdomain.WebHook
type JsonValue = webhookdomain.JsonValue

const hookAuthHeader = "Authorization"

// WebHookAuthentication checks the Authorization header of r against the hook's
// configured secret. Kept here (not on WebHook) because it returns *AppError.
func WebHookAuthentication(w *WebHook, r *http.Request) *AppError {
	if w.Authorization == nil {
		return nil
	}

	res := r.Header.Get(hookAuthHeader)
	if res != *w.Authorization {
		return NewAppError("WebHook", "hook.authentication.unauthorized", nil, "Unauthorized", http.StatusUnauthorized)
	}

	return nil
}
