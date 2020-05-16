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

const (
	conversationCreatedEvent = "created"
	conversationPostedEvent  = "posted"
	conversationJoinedEvent  = "joined"
	conversationClosedEvent  = "closed"
)

type Conversation struct {
	Id         int64     `json:"id" db:"id"`
	Title      string    `json:"title" db:"title"`
	CreatedAt  int64     `json:"created_at" db:"created_at"`
	ActivityAt int64     `json:"activity_at" db:"activity_at"`
	ClosedAt   int64     `json:"closed_at" db:"closed_at"`
	Variables  Variables `json:"variables" db:"variables"`
}

type CreateConversation struct {
	Key   string   `json:"key"`
	Title string   `json:"title"`
	Name  string   `json:"name"`
	Body  PostBody `json:"body"`
}

type PostBody struct {
	Type string          `json:"type"`
	Data json.RawMessage `json:"data"`
}

func (p *PostBody) ToJson() []byte {
	data, _ := json.Marshal(p)
	return data
}

type ConversationInfo struct {
	Id         int64  `json:"id" db:"id"`
	ChannelId  string `json:"channel_id" db:"channel_id"`
	ActivityAt int64  `json:"activity_at" db:"activity_at"`
	Title      string `json:"title" db:"title"`
}

func (c *ConversationInfo) ToJson() []byte {
	data, _ := json.Marshal(c)
	return data
}

type JoinConversation struct {
	Name     string `json:"name"`
	ParentId string `json:"channel_id"` // FIXME
	Body     string `json:"body"`
}

type ConversationMessage struct {
	PostedAt int64    `json:"posted_at" db:"posted_at"`
	PostedBy string   `json:"posted_by" db:"posted_by"`
	Body     PostBody `json:"body" db:"body"`
}

type ConversationMessageJoined struct { //TODO
	ConversationMessage
	ChannelId string `json:"channel_id" db:"channel_id"`
}

func (c *CreateConversation) IsValid() *AppError {
	return nil
}

type ConversationChannel struct {
	ConversationInfo
	WelcomeText string `json:"welcome_text"`
}

func (c *ConversationChannel) ToJson() []byte {
	data, _ := json.Marshal(&c)
	return data
}

type JoinConversationRequest struct {
	ParentChannelId string `json:"parent_channel_id"`
	Name            string `json:"name"`
}

func CreateConversationFromJson(data io.Reader) *CreateConversation {
	var out *CreateConversation
	if err := json.NewDecoder(data).Decode(&out); err == nil {
		return out
	} else {
		return nil
	}
}

func CreateJoinConversationRequestFromJson(data io.Reader) *JoinConversationRequest {
	var out *JoinConversationRequest
	if err := json.NewDecoder(data).Decode(&out); err == nil {
		return out
	} else {
		return nil
	}
}

func ConversationPostMessageFromJson(data io.Reader) *PostBody {
	var out *PostBody
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

func ConversationMessageJoinedListToJson(list []*ConversationMessageJoined) []byte {
	b, _ := json.Marshal(list)
	return b
}
