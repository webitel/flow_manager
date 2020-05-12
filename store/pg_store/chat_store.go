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

// FIXME change to procedure
func (s SqlChatStore) CreateConversation(secretKey string, title string, name string, message string) (string, *model.AppError) {
	channelId, err := s.GetMaster().SelectStr(`with conv as (
    insert into cc_msg_conversation (title, domain_id)
    values (:Title, (select p.domain_id
			from cc_msg_profiles p
			where p.secret_key = :Key))
    returning id
),
part as (
    insert into cc_msg_participants (name, conversation_id)
    select :Name, conv.id
    from conv
    returning channel_id, conversation_id, name
),
post as (
    insert into cc_msg_post(conversation_id, body, posted_by)
    select p.conversation_id, :Message, p.name
    from part p
)
select part.channel_id
from part`, map[string]interface{}{
		"Title":   title,
		"Key":     secretKey,
		"Name":    name,
		"Message": message,
	})

	if err != nil {
		return "", model.NewAppError("SqlChatStore.CreateConversation", "store.sql_chat.create_conversation.error", nil,
			err.Error(), extractCodeFromErr(err))
	}

	return channelId, nil
}

func (s SqlChatStore) ConversationPostMessage(channelId string, body string) ([]*model.ConversationMessage, *model.AppError) {
	var out []*model.ConversationMessage
	_, err := s.GetMaster().Select(&out, `select *
from cc_msg_post(:ChannelId, :Body)  as msg  (posted_at int8, posted_by varchar, body varchar)`, map[string]interface{}{
		"ChannelId": channelId,
		"Body":      body,
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
with part as (
    update cc_msg_participants n
        set last_activity_at  = now()
    from cc_msg_participants o
    where n.channel_id = :ChannelId and n.channel_id = o.channel_id
    returning o.conversation_id, o.last_activity_at
)
select msg.posted_by, (extract(epoch from msg.posted_at) * 1000)::int8 posted_at, msg.body
from part,
     lateral (
        select *
        from cc_msg_post p
        where p.conversation_id = part.conversation_id
            and p.posted_at > part.last_activity_at
        order by p.posted_at desc
        limit :Limit
) msg`, map[string]interface{}{
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
	_, err := s.GetMaster().Select(&msgs, `with part as (
    update cc_msg_participants n
        set last_activity_at  = now()
    from cc_msg_participants o
    where n.channel_id = :ChannelId and n.channel_id = o.channel_id
    returning o.conversation_id, o.last_activity_at
)
select msg.posted_by, (extract(epoch from msg.posted_at) * 1000)::int8 posted_at, msg.body
from part,
     lateral (
        select *
        from cc_msg_post p
        where p.conversation_id = part.conversation_id
        order by p.posted_at desc
        limit :Limit
		offset :Offset
) msg`, map[string]interface{}{
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
