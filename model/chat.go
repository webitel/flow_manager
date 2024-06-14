package model

const (
	// TODO
	ConversationStartMessageVariable = "start_message"
	ConversationSessionId            = "uuid"
	ConversationProfileId            = "wbt_profile_id"

	BreakChatTransferCause = "TRANSFER"
)

type ChatAction string

const (
	ChatActionTyping ChatAction = "typing"
	ChatActionCancel ChatAction = "cancel"
)

type ChatButton struct {
	Text string `json:"text"`
	Type string `json:"type"`
	Url  string `json:"url"`
	Code string `json:"code"`
}

type ChatMenuArgs struct {
	Type    string         `json:"type"` // type
	Buttons [][]ChatButton `json:"buttons"`
	Inline  [][]ChatButton `json:"inline"`
	Text    string         `json:"text"`
}

type ChatMessageOutbound struct {
	Type    string
	Text    string
	File    *File
	Server  string         `json:"server" db:"-"` // TODO
	Buttons [][]ChatButton `json:"buttons"`
	Inline  [][]ChatButton `json:"inline"`
}

type BroadcastChat struct {
	Type    string
	Profile struct {
		Id int64
	}
	Peer         []string
	Text         string
	Menu         *ChatMenuArgs
	File         *File
	Variables    map[string]string
	DomainId     int64
	ResponseCode string `json:"responseCode"`
}

type ChatMessage struct {
	Text       string `json:"text,omitempty" db:"msg"`
	CreatedAt  string `json:"created_at,omitempty" db:"created_at"`
	Type       string `json:"type,omitempty" db:"type"`
	User       string `json:"user,omitempty" db:"name"`
	IsInternal bool   `json:"isInternal" db:"internal"`
}
