package web_chat

import (
	"github.com/webitel/flow_manager/model"
	"net/http"
)

func (s *server) InitApi() {
	s.Router = s.RootRouter.PathPrefix("/").Subrouter()
	media := s.Router.PathPrefix("/chat").Subrouter()
	media.Handle("/conversation", s.ApiHandler(createChat)).Methods("POST")
	media.Handle("/conversation/{id}", s.ApiHandler(listNewMessages)).Methods("GET")
	media.Handle("/conversation/{id}", s.ApiHandler(postChat)).Methods("POST")
	media.Handle("/conversation/{id}/history", s.ApiHandler(historyChat)).Methods("GET")
}

func createChat(c *Context, w http.ResponseWriter, r *http.Request) {
	channel := &model.ConversationChannel{}
	conv := model.CreateConversationFromJson(r.Body)
	defer r.Body.Close()

	if c.Err = conv.IsValid(); c.Err != nil {
		return
	}

	channel.ChannelId, channel.WelcomeText, c.Err = c.app.CreateConversation(conv)

	if c.Err != nil {
		return
	}

	w.WriteHeader(http.StatusCreated)
	w.Write(channel.ToJson())
}

func listNewMessages(c *Context, w http.ResponseWriter, r *http.Request) {
	var msgs []*model.ConversationMessage
	msgs, c.Err = c.app.ConversationUnreadMessages(c.Params.Id, c.Params.PerPage)
	if c.Err != nil {
		return
	}

	w.Write(model.ConversationMessageListToJson(msgs))
}

func postChat(c *Context, w http.ResponseWriter, r *http.Request) {
	conv := model.ConversationPostMessageFromJson(r.Body)
	defer r.Body.Close()
	var msgs []*model.ConversationMessage

	msgs, c.Err = c.app.ConversationPostMessage(c.Params.Id, conv.Body)

	if c.Err != nil {
		return
	}
	w.WriteHeader(http.StatusCreated)
	w.Write(model.ConversationMessageListToJson(msgs))
}

func historyChat(c *Context, w http.ResponseWriter, r *http.Request) {
	var msgs []*model.ConversationMessage

	msgs, c.Err = c.app.ConversationHistory(c.Params.Id, c.Params.PerPage, c.Params.Page)

	if c.Err != nil {
		return
	}
	w.WriteHeader(http.StatusCreated)
	w.Write(model.ConversationMessageListToJson(msgs))
}
