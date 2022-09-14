package model

import "time"

type ListCommunication struct {
	Destination string     `json:"destination"`
	Description *string    `json:"description"`
	ExpireAt    *time.Time `json:"expire_at"`
}
