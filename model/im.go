package model

import "context"

type IMDialog interface {
	Connection
	SchemaId() int
	Stop(err error)
	IsTransfer() bool
	SendMessage(ctx context.Context, msg ChatMessageOutbound) (Response, *AppError)
	SendTextMessage(ctx context.Context, text string) (Response, *AppError)
	ReceiveMessage(ctx context.Context, name string, timeout, messageTimeout int) ([]string, *AppError)
}

// MessageWrapper представляє кореневий об'єкт
type MessageWrapper struct {
	ID       string  `json:"id"`
	Message  Message `json:"payload"`
	UserID   string  `json:"user_id"`
	DomainID int64   `json:"domain_id"`
	Echo     bool    `json:"echo"`
}

// Message описує вкладений об'єкт повідомлення
type Message struct {
	ID        string     `json:"ID"`
	ThreadID  string     `json:"ThreadID"`
	DomainID  int        `json:"DomainID"`
	From      ImEndpoint `json:"From"`
	To        ImEndpoint `json:"To"`
	Text      string     `json:"Text"`
	CreatedAt int64      `json:"CreatedAt"` // Unix timestamp у мілісекундах
}

// From описує відправника
type ImEndpoint struct {
	ID     string `json:"id"`
	Type   int    `json:"type"`
	Sub    string `json:"sub"`
	Issuer string `json:"issuer"`
	Name   string `json:"name"`
}
