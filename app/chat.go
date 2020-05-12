package app

import "github.com/webitel/flow_manager/model"

func (fm *FlowManager) CreateConversation(con *model.CreateConversation) (string, string, *model.AppError) {
	conv, err := fm.Store.Chat().CreateConversation(con.Key, con.Title, con.Name, con.Body)
	if err != nil {

		return "", "", err
	}

	if text, e := fm.cc.Member().JoinChatToQueue(1, conv, 15, con.Name, con.Title); e != nil {
		return "", "", model.NewAppError("CC", "cc", nil, e.Error(), 500)
	} else {
		return conv, text, nil
	}
}

func (fm *FlowManager) ConversationUnreadMessages(channelId string, limit int) ([]*model.ConversationMessage, *model.AppError) {
	return fm.Store.Chat().ConversationUnreadMessages(channelId, limit)
}

func (fm *FlowManager) ConversationPostMessage(channelId string, body string) ([]*model.ConversationMessage, *model.AppError) {
	return fm.Store.Chat().ConversationPostMessage(channelId, body)
}

func (fm *FlowManager) ConversationHistory(channelId string, limit, offset int) ([]*model.ConversationMessage, *model.AppError) {
	return fm.Store.Chat().ConversationHistory(channelId, limit, offset)
}

func (fm *FlowManager) PushChatToQueue(domainId int64, channelId string, queueId int64, name, number string) (string, error) {
	return fm.cc.Member().JoinChatToQueue(domainId, channelId, queueId, name, number)
}
