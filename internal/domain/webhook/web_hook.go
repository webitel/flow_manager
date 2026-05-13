package webhook

// moved from model/web_hook.go — see model/web_hook.go for re-export aliases

import (
	"encoding/json"
	"strings"
)

// WebHook describes an inbound webhook configuration.
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

// JsonValue is a raw JSON value.
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

func (w *WebHook) InitOrigin() {
	w.allowedOrigins = make([]originPattern, 0, len(w.AllowOrigins))
	for _, origin := range w.AllowOrigins {
		origin = strings.ToLower(origin)
		if origin == "*" {
			w.allowedOrigins = append(w.allowedOrigins[:0], originAny(true))
			break
		} else if i := strings.IndexByte(origin, '*'); i >= 0 {
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
