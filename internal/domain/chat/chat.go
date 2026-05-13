package chat

// moved from model/chat.go — see model/chat.go for re-export aliases

import "github.com/webitel/flow_manager/internal/domain/files"

// ChatAction is the type of a transient chat action (e.g. "typing").
type ChatAction string

const (
	ChatActionTyping ChatAction = "typing"
	ChatActionCancel ChatAction = "cancel"
)

// ChatButton is a single interactive button in a chat message.
type ChatButton struct {
	Caption string `json:"caption"`
	Text    string `json:"text"`
	Type    string `json:"type"`
	Url     string `json:"url"`
	Code    string `json:"code"`
}

// ChatMenuArgs describes a menu payload sent to a chat client.
type ChatMenuArgs struct {
	Type    string         `json:"type"`
	Buttons [][]ChatButton `json:"buttons"`
	Inline  [][]ChatButton `json:"inline"`
	Text    string         `json:"text"`
	NoInput bool           `json:"noInput"`
	Kind    string         `json:"kind"`
}

// ChatMessageOutbound is an outbound chat message payload.
type ChatMessageOutbound struct {
	Type    string
	Text    string
	File    *files.File
	Server  string         `json:"server" db:"-"` // TODO
	Buttons [][]ChatButton `json:"buttons"`
	Inline  [][]ChatButton `json:"inline"`
	NoInput bool           `json:"noInput"`
	Kind    string         `json:"kind"`
}

// BroadcastPeer is a single recipient in a broadcast chat.
type BroadcastPeer struct {
	Id   string `json:"id,omitempty"`
	Type string `json:"type,omitempty"`
	Via  string `json:"via,omitempty"`
}

// BroadcastChat is the request payload for a broadcast chat message.
type BroadcastChat struct {
	Type    any
	Profile struct {
		Id int
	}
	Peer    []any
	Text    string
	File    *files.File
	Buttons [][]ChatButton `json:"buttons"`

	Variables    map[string]string
	DomainId     int64
	ResponseCode string `json:"responseCode"`
	// FailedReceivers used to set the variable name in which failed receivers are saved.
	FailedReceivers string `json:"failedReceivers"`
	// Timeout determines how long chat_manager waits for the host callback (in secs).
	Timeout int64 `json:"timeout"`
}

// FailedReceiver is a single failed broadcast recipient.
type FailedReceiver struct {
	Id    string `json:"id,omitempty"`
	Error string `json:"error,omitempty"`
}

// BroadcastChatResponse is the result of a broadcast chat operation.
type BroadcastChatResponse struct {
	Failed    []*FailedReceiver `json:"failed"`
	Variables map[string]string
}

// ChatMessage is a single message retrieved from conversation history.
type ChatMessage struct {
	Text       string `json:"text,omitempty" db:"msg"`
	CreatedAt  string `json:"created_at,omitempty" db:"created_at"`
	Type       string `json:"type,omitempty" db:"type"`
	User       string `json:"user,omitempty" db:"name"`
	IsInternal bool   `json:"isInternal" db:"internal"`
}
