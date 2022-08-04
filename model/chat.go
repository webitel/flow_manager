package model

const (
	// TODO
	ConversationStartMessageVariable = "start_message"
	ConversationSessionId            = "uuid"
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
	Set     string         `json:"set"`
}

type BroadcastChat struct {
	Type    string
	Profile struct {
		Id int64
	}
	Peer      []string
	Text      string
	Menu      *ChatMenuArgs
	File      *File
	Variables map[string]string
	DomainId  int64
}
