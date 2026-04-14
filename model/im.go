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
	SetQueue(*InQueueKey) bool
	GetQueueKey() *InQueueKey
}

type CCQueueEvent struct {
	AttemptId int64  `json:"attempt_id"`
	Event     string `json:"event"`
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
	ID        string     `json:"id"`
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
