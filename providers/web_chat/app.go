package web_chat

import "github.com/webitel/flow_manager/model"

type App interface {
	CreateConversation(con *model.CreateConversation) (string, string, *model.AppError)
	ConversationUnreadMessages(channelId string, limit int) ([]*model.ConversationMessage, *model.AppError)
	ConversationPostMessage(channelId string, body string) ([]*model.ConversationMessage, *model.AppError)
	ConversationHistory(channelId string, limit, offset int) ([]*model.ConversationMessage, *model.AppError)
	PushChatToQueue(domainId int64, channelId string, queueId int64, name, number string) (string, error)
}
