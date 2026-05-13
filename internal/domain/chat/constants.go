package chat

// moved from model/chat.go

const (
	// ConversationStartMessageVariable is the flow variable that holds the first
	// message of a chat/IM conversation.
	ConversationStartMessageVariable = "start_message"
	ConversationSessionId            = "uuid"
	ConversationProfileId            = "wbt_profile_id"

	BreakChatTransferCause = "TRANSFER"
)
