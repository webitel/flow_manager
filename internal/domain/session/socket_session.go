package session

// moved from model/socket_session.go — see model/socket_session.go for re-export alias

import "time"

// SocketSession represents an active WebSocket session.
type SocketSession struct {
	ID              string    `json:"id" db:"id"`
	CreatedAt       time.Time `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time `json:"updated_at" db:"updated_at"`
	UserAgent       string    `json:"user_agent" db:"user_agent"`
	UserID          int64     `json:"user_id" db:"user_id"`
	AppID           string    `json:"app_id" db:"app_id"`
	ApplicationName string    `json:"application_name" db:"application_name"`
}
