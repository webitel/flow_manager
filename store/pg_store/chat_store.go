package sqlstore

import (
	"github.com/webitel/flow_manager/model"
	"github.com/webitel/flow_manager/store"
)

type SqlChatStore struct {
	SqlStore
}

func NewSqlChatStore(sqlStore SqlStore) store.ChatStore {
	st := &SqlChatStore{sqlStore}
	return st
}

func (s SqlChatStore) CreateConversation(secretKey string, title string, name string, body model.PostBody) (model.ConversationInfo, *model.AppError) {
	var info model.ConversationInfo
	err := s.GetMaster().SelectOne(&info, `select call_center.cc_view_timestamp(x.posted_at) activity_at,
       x.id, x.channel_id
from call_center.cc_msg_create_conversation(:Key, :Title, :Name, :Body)
as x(posted_at timestamptz, id int8, channel_id text);`, map[string]interface{}{
		"Title": title,
		"Key":   secretKey,
		"Name":  name,
		"Body":  body.ToJson(),
	})

	if err != nil {
		return info, model.NewAppError("SqlChatStore.CreateConversation", "store.sql_chat.create_conversation.error", nil,
			err.Error(), extractCodeFromErr(err))
	}

	return info, nil
}

func (s SqlChatStore) Get(channelId string) (*model.ConversationInfo, *model.AppError) {
	var info *model.ConversationInfo
	err := s.GetReplica().SelectOne(&info, `select cmc.id, part.channel_id, cc_view_timestamp(cmc.activity_at) activity_at, cmc.title
from call_center.cc_msg_participants part
    inner join call_center.cc_msg_conversation cmc on part.conversation_id = cmc.id
where part.channel_id = :ChannelId and cmc.closed_at is null`, map[string]interface{}{
		"ChannelId": channelId,
	})

	if err != nil {
		return info, model.NewAppError("SqlChatStore.Get", "store.sql_chat.get_conversation.error", nil,
			err.Error(), extractCodeFromErr(err))
	}

	return info, nil
}

func (s SqlChatStore) ConversationPostMessage(channelId string, body model.PostBody) ([]*model.ConversationMessage, *model.AppError) {
	var out []*model.ConversationMessage
	_, err := s.GetMaster().Select(&out, `select *
from call_center.cc_msg_post(:ChannelId, :Body)  as msg  (posted_at int8, posted_by varchar, body jsonb)`, map[string]interface{}{
		"ChannelId": channelId,
		"Body":      body.ToJson(),
	})

	if err != nil {
		return nil, model.NewAppError("SqlChatStore.ConversationPostMessage", "store.sql_chat.post_message.error", nil,
			err.Error(), extractCodeFromErr(err))
	}

	return out, nil
}

func (s SqlChatStore) ConversationUnreadMessages(channelId string, limit int) ([]*model.ConversationMessage, *model.AppError) {
	var msgs []*model.ConversationMessage
	_, err := s.GetReplica().Select(&msgs, `
select *
from call_center.cc_msg_unread(:ChannelId, :Limit)
    as x (posted_by varchar, posted_at int8, body jsonb)`, map[string]interface{}{
		"ChannelId": channelId,
		"Limit":     limit,
	})

	if err != nil {
		return nil, model.NewAppError("SqlChatStore.ConversationUnreadMessages", "store.sql_chat.last_msg.error", nil,
			err.Error(), extractCodeFromErr(err))
	}

	return msgs, nil

}

func (s SqlChatStore) ConversationHistory(channelId string, limit, offset int) ([]*model.ConversationMessage, *model.AppError) {
	var msgs []*model.ConversationMessage
	_, err := s.GetMaster().Select(&msgs, `select *
from call_center.cc_msg_history(:ChannelId, :Limit, :Offset)
    as x (posted_by varchar, posted_at int8, body jsonb)`, map[string]interface{}{
		"ChannelId": channelId,
		"Limit":     limit,
		"Offset":    offset * limit,
	})

	if err != nil {
		return nil, model.NewAppError("SqlChatStore.ConversationHistory", "store.sql_chat.history_msg.error", nil,
			err.Error(), extractCodeFromErr(err))
	}

	return msgs, nil
}

func (s SqlChatStore) Join(parentChannelId string, name string) ([]*model.ConversationMessageJoined, *model.AppError) {
	var msgs []*model.ConversationMessageJoined
	_, err := s.GetMaster().Select(&msgs, `with part as (
    insert into call_center.cc_msg_participants (name, conversation_id)
    select :Name, p.conversation_id
    from call_center.cc_msg_participants p
    where p.channel_id = :ChannelId
    returning channel_id, conversation_id, name
)
select  (extract(epoch from msg.posted_at) * 1000)::int8 posted_at, msg.posted_by, msg.body, part.channel_id
from part,
     lateral (
        select *
        from call_center.cc_msg_post p
        where p.conversation_id = part.conversation_id
        order by p.posted_at desc
        limit 20
    ) msg`, map[string]interface{}{
		"ChannelId": parentChannelId,
		"Name":      name,
	})

	if err != nil {
		return nil, model.NewAppError("SqlChatStore.ConversationHistory", "store.sql_chat.join.error", nil,
			err.Error(), extractCodeFromErr(err))
	}

	return msgs, nil
}

func (s SqlChatStore) Close(channelId string) *model.AppError {
	_, err := s.GetMaster().Exec(`update call_center.cc_msg_conversation c
set closed_at = now(),
    closed_by = part.id
from (
    select part.id, part.conversation_id
    from call_center.cc_msg_participants part
    where part.channel_id = :ChannelId
) part
where part.conversation_id = c.id and c.closed_at is null`, map[string]interface{}{
		"ChannelId": channelId,
	})

	if err != nil {
		return model.NewAppError("SqlChatStore.ConversationHistory", "store.sql_chat.join.error", nil,
			err.Error(), extractCodeFromErr(err))
	}

	return nil
}
