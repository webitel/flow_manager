package list

// moved from model/list.go — see model/list.go for re-export alias

import "time"

// ListCommunication is a communication entry stored in a black/white-list.
type ListCommunication struct {
	Destination string     `json:"destination"`
	Description *string    `json:"description"`
	ExpireAt    *time.Time `json:"expire_at"`
}
