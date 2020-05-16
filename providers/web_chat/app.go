package web_chat

import "github.com/webitel/flow_manager/model"

type App interface {
	GetConversation(channelId string) (*model.ConversationInfo, *model.AppError)
	CreateConversation(con *model.CreateConversation) (model.ConversationInfo, string, *model.AppError)
	ConversationUnreadMessages(channelId string, limit int) ([]*model.ConversationMessage, *model.AppError)
	ConversationPostMessage(channelId string, body model.PostBody) ([]*model.ConversationMessage, *model.AppError)
	ConversationHistory(channelId string, limit, offset int) ([]*model.ConversationMessage, *model.AppError)
	PushChatToQueue(domainId int64, channelId string, queueId int64, name, number string) (string, error)
	JoinToConversation(parentChannelId string, name string) ([]*model.ConversationMessageJoined, *model.AppError)
	CloseConversation(channelId string) *model.AppError
}
