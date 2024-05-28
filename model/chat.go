package model

const (
	// TODO
	ConversationStartMessageVariable = "start_message"
	ConversationSessionId            = "uuid"

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
	// FailedReceivers used to set the variable name in which will be saved failed receivers. (if not set then info about failed receivers will not be saved)
	FailedReceivers string `json:"failedReceivers"`
	// Timeout determines how much time chat_manager is waiting (in secs) for the response from the host(telegram|whatsapp..) on our callback url.
	Timeout int64 `json:"timeout"`
}

type BroadcastChatResponse struct {
	Failed    []*FailedReceiver `json:"failed"`
	Variables map[string]string
}

type FailedReceiver struct {
	Id    string `json:"id,omitempty"`
	Error string `json:"error,omitempty"`
}

type ChatMessage struct {
	Text       string `json:"text,omitempty" db:"msg"`
	CreatedAt  string `json:"created_at,omitempty" db:"created_at"`
	Type       string `json:"type,omitempty" db:"type"`
	User       string `json:"user,omitempty" db:"name"`
	IsInternal bool   `json:"isInternal" db:"internal"`
}
