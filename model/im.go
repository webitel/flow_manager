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
	ID       string  `json:"ID"`
	Message  Message `json:"message"`
	UserID   string  `json:"user_id"`
	DomainID int64   `json:"domain_id"`
}

// Message описує вкладений об'єкт повідомлення
type Message struct {
	ID        string     `json:"ID"`
	ThreadID  string     `json:"thread_id"`
	DomainID  int        `json:"domain_id"`
	From      ImEndpoint `json:"from"`
	To        ImEndpoint `json:"to"`
	Text      string     `json:"text"`
	CreatedAt int64      `json:"created_at"` // Unix timestamp у мілісекундах
}

// From описує відправника
type ImEndpoint struct {
	ID     string `json:"id"`
	Type   int    `json:"type"`
	Sub    string `json:"sub"`
	Issuer string `json:"issuer"`
	Name   string `json:"name"`
}
