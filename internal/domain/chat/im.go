package chat

// moved from model/im.go — see model/im.go for re-export aliases

// CCQueueEvent is an event received from the call-centre queue.
type CCQueueEvent struct {
	AttemptId int64  `json:"attempt_id"`
	Event     string `json:"event"`
	Result    string `json:"result"`
}

// MessageWrapper is the root object for an incoming IM message event.
type MessageWrapper struct {
	ID       string  `json:"id"`
	Message  Message `json:"payload"`
	UserID   string  `json:"user_id"`
	DomainID int64   `json:"domain_id"`
	Echo     bool    `json:"echo"`
}

// Message is the nested message payload within a MessageWrapper.
type Message struct {
	ID        string       `json:"id"`
	ThreadID  string       `json:"thread_id"`
	DomainID  int          `json:"domain_id"`
	From      ImEndpoint   `json:"from"`
	To        []ImEndpoint `json:"to"`
	Text      string       `json:"text"`
	CreatedAt int64        `json:"created_at"` // Unix timestamp in milliseconds
}

// ImEndpoint describes a participant in an IM conversation.
type ImEndpoint struct {
	ID     string `json:"id"`
	Type   int    `json:"type"`
	Sub    string `json:"sub"`
	Issuer string `json:"issuer"`
	Name   string `json:"name"`
}
