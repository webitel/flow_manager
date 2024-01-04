package model

import (
	"encoding/json"
	"net/http"
	"strings"
)

const (
	hookAuthHeader = "Authorization"
)

type WebHook struct {
	Key            string   `json:"key" db:"key"`
	Name           string   `json:"name" db:"name"`
	Enabled        bool     `json:"enabled" db:"enabled"`
	SchemaId       int      `json:"schema_id" db:"schema_id"`
	AllowOrigins   []string `json:"origin" db:"origin"`
	DomainId       int64    `json:"domain_id" db:"domain_id"`
	Authorization  *string  `json:"authorization" db:"authorization"`
	allowedOrigins []originPattern
}

type JsonValue json.RawMessage

type originPattern interface {
	match(origin string) bool
}

type originAny bool

func (pttn originAny) match(origin string) bool {
	return origin != "" && (bool)(pttn)
}

type originWildcard [2]string

func (pttn originWildcard) match(origin string) bool {
	prefix, suffix := pttn[0], pttn[1]
	return len(origin) >= len(prefix)+len(suffix) &&
		strings.HasPrefix(origin, prefix) &&
		strings.HasSuffix(origin, suffix)
}

type originString string

func (pttn originString) match(origin string) bool {
	return (string)(pttn) == (origin)
}

func (w *WebHook) Authentication(r *http.Request) *AppError {
	if w.Authorization == nil {
		return nil
	}

	res := r.Header.Get(hookAuthHeader)
	// TODO any auth type
	if res != *w.Authorization {
		return NewAppError("WebHook", "hook.authentication.unauthorized", nil, "Unauthorized", http.StatusUnauthorized)
	}

	return nil
}

func (w *WebHook) InitOrigin() {
	w.allowedOrigins = make([]originPattern, 0, len(w.AllowOrigins))
	for _, origin := range w.AllowOrigins {
		// Normalize
		origin = strings.ToLower(origin)
		if origin == "*" {
			// If "*" is present in the list, turn the whole list into a match all
			w.allowedOrigins = append(w.allowedOrigins[:0], originAny(true))
			break
		} else if i := strings.IndexByte(origin, '*'); i >= 0 {
			// Split the origin in two: start and end string without the *
			w.allowedOrigins = append(w.allowedOrigins, originWildcard{origin[0:i], origin[i+1:]})
		} else if origin != "" {
			w.allowedOrigins = append(w.allowedOrigins, originString(origin))
		}
	}
}

func (w *WebHook) AllowOrigin(origin string) bool {
	if len(w.allowedOrigins) != 0 {
		origin = strings.ToLower(origin)
		for _, allowedOrigin := range w.allowedOrigins {
			if allowedOrigin.match(origin) {
				return true
			}
		}
		return false
	}

	return true
}
