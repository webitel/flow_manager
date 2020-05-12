package model

import (
	"encoding/json"
	"io"
)

//type ChannelCreated struct {
//	ChannelId      string `json:"channel_id" db:"channel_id"`
//	ConversationId int64  `json:"conversation_id" db:"conversation_id"`
//	TimeoutAt      int64  `json:"timeout_at" db:"timeout_at"`
//}

type CreateConversation struct {
	Key   string `json:"key"`
	Title string `json:"title"`
	Name  string `json:"name"`
	Body  string `json:"body"`
}

type ConversationMessage struct {
	PostedAt int64  `json:"posted_at" db:"posted_at"`
	PostedBy string `json:"posted_by" db:"posted_by"`
	Body     string `json:"body" db:"body"`
}

type ConversationPostMessage struct {
	Body string `json:"body" db:"body"`
}

func (c *CreateConversation) IsValid() *AppError {
	return nil
}

type ConversationChannel struct {
	ChannelId   string `json:"channel_id"`
	WelcomeText string `json:"welcome_text"`
}

func (c *ConversationChannel) ToJson() []byte {
	data, _ := json.Marshal(&c)
	return data
}

func CreateConversationFromJson(data io.Reader) *CreateConversation {
	var out *CreateConversation
	if err := json.NewDecoder(data).Decode(&out); err == nil {
		return out
	} else {
		return nil
	}
}

func ConversationPostMessageFromJson(data io.Reader) *ConversationPostMessage {
	var out *ConversationPostMessage
	if err := json.NewDecoder(data).Decode(&out); err == nil {
		return out
	} else {
		return nil
	}
}

func ConversationMessageListToJson(list []*ConversationMessage) []byte {
	b, _ := json.Marshal(list)
	return b
}
