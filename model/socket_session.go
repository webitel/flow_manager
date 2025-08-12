package model

import "time"

type SocketSession struct {
	ID              string    `json:"id" db:"id"`
	CreatedAt       time.Time `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time `json:"updated_at" db:"updated_at"`
	UserAgent       string    `json:"user_agent" db:"user_agent"`
	UserID          int64     `json:"user_id" db:"user_id"`
	AppID           string    `json:"app_id" db:"app_id"`
	ApplicationName string    `json:"application_name" db:"application_name"`
}
