package web_chat

import (
	"encoding/json"
	"github.com/webitel/flow_manager/model"
	"net/http"
)

func (s *server) InitApi() {
	s.Router = s.RootRouter.PathPrefix("/").Subrouter()
	media := s.Router.PathPrefix("/chat").Subrouter()
	media.Handle("/conversation", s.ApiHandler(createChat)).Methods("POST")
	media.Handle("/conversation/{id}", s.ApiHandler(getConversation)).Methods("GET")

	media.Handle("/conversation/{id}/unread", s.ApiHandler(listNewMessages)).Methods("GET")
	media.Handle("/conversation/{id}", s.ApiHandler(postChat)).Methods("POST")
	media.Handle("/conversation/{id}", s.ApiHandler(closeConversation)).Methods("DELETE")
	media.Handle("/conversation/{id}/join", s.ApiHandler(join)).Methods("POST")
	media.Handle("/conversation/{id}/history", s.ApiHandler(historyChat)).Methods("GET")
	media.Handle("/fb", s.ApiHandler(fb)).Methods("GET")
}

///chat/fb?hub.mode=subscribe&hub.challenge=789336384&hub.verify_token=ddd
func fb(c *Context, w http.ResponseWriter, r *http.Request) {
	req := make(map[string]interface{})
	json.NewDecoder(r.Body).Decode(&req)

	w.WriteHeader(http.StatusOK)
	token := r.URL.Query().Get("hub.challenge")

	w.Write([]byte(token))
}
func getConversation(c *Context, w http.ResponseWriter, r *http.Request) {
	var info *model.ConversationInfo
	info, c.Err = c.app.GetConversation(c.Params.Id)
	if c.Err != nil {
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write(info.ToJson())
}

func closeConversation(c *Context, w http.ResponseWriter, r *http.Request) {
	c.Err = c.app.CloseConversation(c.Params.Id)
	if c.Err != nil {
		return
	}

	w.WriteHeader(http.StatusOK)
}

func join(c *Context, w http.ResponseWriter, r *http.Request) {

	req := model.CreateJoinConversationRequestFromJson(r.Body)
	defer r.Body.Close()

	var list []*model.ConversationMessageJoined
	list, c.Err = c.app.JoinToConversation(c.Params.Id, req.Name)
	if c.Err != nil {
		return
	}

	w.WriteHeader(http.StatusCreated)
	w.Write(model.ConversationMessageJoinedListToJson(list))
}

func createChat(c *Context, w http.ResponseWriter, r *http.Request) {
	channel := &model.ConversationChannel{}
	conv := model.CreateConversationFromJson(r.Body)
	defer r.Body.Close()

	if c.Err = conv.IsValid(); c.Err != nil {
		return
	}

	channel.ConversationInfo, channel.WelcomeText, c.Err = c.app.CreateConversation(conv)

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
	post := model.ConversationPostMessageFromJson(r.Body)
	defer r.Body.Close()
	var msgs []*model.ConversationMessage

	msgs, c.Err = c.app.ConversationPostMessage(c.Params.Id, *post)

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
	w.WriteHeader(http.StatusOK)
	w.Write(model.ConversationMessageListToJson(msgs))
}
