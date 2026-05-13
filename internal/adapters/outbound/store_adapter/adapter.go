// Package store_adapter wraps store.Store and exposes the thin delegating
// methods that used to live in app/. Each method is a single call to the
// appropriate Store sub-interface, with error conversion to *model.AppError.
package store_adapter

import (
	"context"
	"net/http"
	"time"

	"github.com/webitel/flow_manager/internal/infrastructure/cache"
	"github.com/webitel/flow_manager/model"
	"github.com/webitel/flow_manager/store"
)

// Adapter wraps a store.Store and provides thin delegating methods.
// Embed *Adapter in FlowManager to promote all methods without re-declaring
// them one by one in app/.
type Adapter struct {
	store         store.Store
	externalStore *cache.ExternalStoreManager // optional; set via SetExternalStore
}

// New creates a new Adapter backed by s.
func New(s store.Store) *Adapter {
	return &Adapter{store: s}
}

// toAppError converts a plain error to *model.AppError; nil in → nil out.
func toAppError(op string, err error) *model.AppError {
	if err == nil {
		return nil
	}
	if ae, ok := err.(*model.AppError); ok {
		return ae
	}
	return model.NewAppError(op, "app.store_err", nil, err.Error(), http.StatusInternalServerError)
}

// ── Media ─────────────────────────────────────────────────────────────────────

func (a *Adapter) GetMediaFiles(domainId int64, req *[]*model.PlaybackFile) ([]*model.PlaybackFile, *model.AppError) {
	res, err := a.store.Media().GetFiles(domainId, req)
	return res, toAppError("App.GetMediaFiles", err)
}

func (a *Adapter) GetMediaFile(domainId int64, id int) (*model.File, *model.AppError) {
	res, err := a.store.Media().Get(domainId, id)
	return res, toAppError("App.GetMediaFile", err)
}

func (a *Adapter) SearchMediaFile(domainId int64, search *model.SearchFile) (*model.File, *model.AppError) {
	res, err := a.store.Media().SearchOne(domainId, search)
	return res, toAppError("App.SearchMediaFile", err)
}

func (a *Adapter) GetPlaybackFile(domainId int64, search *model.PlaybackFile) (*model.PlaybackFile, *model.AppError) {
	res, err := a.store.Media().GetPlaybackFile(domainId, search)
	return res, toAppError("App.GetPlaybackFile", err)
}

// ── Log ───────────────────────────────────────────────────────────────────────

func (a *Adapter) StoreLog(schemaId int, connId string, log []*model.StepLog) *model.AppError {
	if len(log) == 0 {
		return nil
	}
	if err := a.store.Log().Save(schemaId, connId, log); err != nil {
		return model.NewAppError("StoreLog", "store.log.save", nil, err.Error(), http.StatusInternalServerError)
	}
	return nil
}

// ── Queue ─────────────────────────────────────────────────────────────────────

func (a *Adapter) FindQueueByName(domainId int64, name string) (int32, *model.AppError) {
	id, err := a.store.Queue().FindQueueByName(domainId, name)
	if err != nil {
		return 0, model.NewAppError("FindQueueByName", "store.queue.find_by_name", nil, err.Error(), http.StatusInternalServerError)
	}
	return id, nil
}

// ── User ──────────────────────────────────────────────────────────────────────

func (a *Adapter) GetUserProperties(domainId int64, search *model.SearchUser, mapRes model.Variables) (model.Variables, *model.AppError) {
	res, err := a.store.User().GetProperties(domainId, search, mapRes)
	if err != nil {
		return nil, model.NewAppError("GetUserProperties", "store.user.get_properties", nil, err.Error(), http.StatusInternalServerError)
	}
	return res, nil
}

func (a *Adapter) GetAgentIdByExtension(domainId int64, extension string) (*int32, *model.AppError) {
	res, err := a.store.User().GetAgentIdByExtension(domainId, extension)
	if err != nil {
		return nil, model.NewAppError("GetAgentIdByExtension", "store.user.get_agent_id_by_extension", nil, err.Error(), http.StatusInternalServerError)
	}
	return res, nil
}

// ── Call ──────────────────────────────────────────────────────────────────────

func (a *Adapter) SetCallGranteeId(domainId int64, id string, granteeId int64) *model.AppError {
	return toAppError("App.SetCallGranteeId", a.store.Call().SetGranteeId(domainId, id, granteeId))
}

func (a *Adapter) SetBlindTransferNumber(domainId int64, callId, destination string) *model.AppError {
	return toAppError("App.SetBlindTransferNumber", a.store.Call().SetBlindTransfer(domainId, callId, destination))
}

func (a *Adapter) CallSetContactId(domainId int64, callId string, contactId int64) *model.AppError {
	return toAppError("App.CallSetContactId", a.store.Call().SetContactId(domainId, callId, contactId))
}

func (a *Adapter) StoreCallVariables(id string, vars map[string]string) *model.AppError {
	if len(vars) == 0 {
		return nil
	}
	cv := make(model.CallVariables)
	for k, v := range vars {
		cv[k] = v
	}
	return toAppError("App.StoreCallVariables", a.store.Call().SetVariables(id, &cv))
}

func (a *Adapter) UpdateCallFrom(id string, name, number, destination *string) *model.AppError {
	return toAppError("App.UpdateCallFrom", a.store.Call().UpdateFrom(id, name, number, destination))
}

func (a *Adapter) LastBridgedCall(domainId int64, number, hours string, dialer, inbound, outbound *string, queueIds []int, mapRes model.Variables) (model.Variables, *model.AppError) {
	res, err := a.store.Call().LastBridged(domainId, number, hours, dialer, inbound, outbound, queueIds, mapRes)
	return res, toAppError("App.LastBridgedCall", err)
}

func (a *Adapter) SetCallUserId(domainId int64, id string, userId int64) *model.AppError {
	return toAppError("App.SetCallUserId", a.store.Call().SetUserId(domainId, id, userId))
}

// ── Member ────────────────────────────────────────────────────────────────────

func (a *Adapter) GetCallPosition(callId string) (int64, *model.AppError) {
	res, err := a.store.Member().CallPosition(callId)
	return res, toAppError("App.GetCallPosition", err)
}

func (a *Adapter) GetMemberProperties(domainId int64, search *model.SearchMember, mapRes model.Variables) (model.Variables, *model.AppError) {
	res, err := a.store.Member().GetProperties(domainId, search, mapRes)
	return res, toAppError("App.GetMemberProperties", err)
}

func (a *Adapter) PatchMembers(domainId int64, search *model.SearchMember, patch *model.PatchMember) (int, *model.AppError) {
	res, err := a.store.Member().PatchMembers(domainId, search, patch)
	return res, toAppError("App.PatchMembers", err)
}

func (a *Adapter) EWTPuzzle(domainId int64, callId string, min int, queueIds []int, bucketIds []int) (float64, *model.AppError) {
	res, err := a.store.Member().EWTPuzzle(domainId, callId, min, queueIds, bucketIds)
	return res, toAppError("App.EWTPuzzle", err)
}

// ── Calendar ──────────────────────────────────────────────────────────────────

func (a *Adapter) CheckCalendar(domainId int64, id *int, name *string) (*model.Calendar, *model.AppError) {
	c, err := a.store.Calendar().Check(domainId, id, name)
	if err != nil {
		return nil, model.NewAppError("CheckCalendar", "store.calendar.check", nil, err.Error(), http.StatusInternalServerError)
	}
	return c, nil
}

// ── Email ─────────────────────────────────────────────────────────────────────

func (a *Adapter) GetEmailProperties(domainId int64, id *int64, messageId *string, mapRes model.Variables) (model.Variables, *model.AppError) {
	vars, err := a.store.Email().GerProperties(domainId, id, messageId, mapRes)
	if err != nil {
		return nil, model.NewAppError("GetEmailProperties", "store.email.get_properties", nil, err.Error(), http.StatusInternalServerError)
	}
	return vars, nil
}

func (a *Adapter) MailSetContacts(ctx context.Context, domainId int64, id string, contactIds []int64) *model.AppError {
	if err := a.store.Email().SetContact(ctx, domainId, id, contactIds); err != nil {
		return model.NewAppError("MailSetContacts", "store.email.set_contact", nil, err.Error(), http.StatusInternalServerError)
	}
	return nil
}

func (a *Adapter) SaveEmail(domainId int64, email *model.Email) error {
	return a.store.Email().Save(domainId, email)
}

func (a *Adapter) GetFileMetadata(domainId int64, ids []int64) ([]model.File, error) {
	return a.store.File().GetMetadata(domainId, ids)
}

// ── Chat (store-only) ─────────────────────────────────────────────────────────

func (a *Adapter) LastBridgedChat(domainId int64, number, hours string, queueIds []int, mapRes model.Variables) (model.Variables, *model.AppError) {
	vars, err := a.store.Chat().LastBridged(domainId, number, hours, queueIds, mapRes)
	if err != nil {
		return nil, model.NewAppError("LastBridgedChat", "store.chat.last_bridged", nil, err.Error(), http.StatusInternalServerError)
	}
	return vars, nil
}

func (a *Adapter) ChatProfileType(domainId int64, profileId int) (string, error) {
	return a.store.Chat().ProfileType(domainId, profileId)
}

func (a *Adapter) GetChatMessagesByConversationId(ctx context.Context, domainId int64, conversationId string, limit int64) (*[]model.ChatMessage, error) {
	messages, storeErr := a.store.Chat().GetMessagesByConversation(ctx, domainId, conversationId, limit)
	if storeErr != nil {
		return nil, storeErr
	}
	return &messages, nil
}

// ── List (store-only) ─────────────────────────────────────────────────────────

// CheckList satisfies the builtin.ListDeps interface; delegates to ListCheckNumber.
func (a *Adapter) CheckList(domainId int64, number string, listId *int, listName *string) (bool, error) {
	ok, appErr := a.ListCheckNumber(domainId, number, listId, listName)
	if appErr != nil {
		return false, appErr
	}
	return ok, nil
}

// AddToList satisfies the builtin.ListDeps interface; delegates to ListAddCommunication.
func (a *Adapter) AddToList(ctx context.Context, domainId int64, listId *int, listName *string, destination string, description *string, expireAtMS int64) error {
	comm := &model.ListCommunication{
		Destination: destination,
		Description: description,
	}
	if expireAtMS > 0 {
		t := time.UnixMilli(expireAtMS)
		comm.ExpireAt = &t
	}
	return a.ListAddCommunication(domainId, &model.SearchEntity{Id: listId, Name: listName}, comm)
}

func (a *Adapter) ListCheckNumber(domainId int64, number string, listId *int, listName *string) (bool, *model.AppError) {
	ok, err := a.store.List().CheckNumber(domainId, number, listId, listName)
	if err != nil {
		return false, model.NewAppError("ListCheckNumber", "store.list.check_number", nil, err.Error(), http.StatusInternalServerError)
	}
	return ok, nil
}

func (a *Adapter) ListAddCommunication(domainId int64, search *model.SearchEntity, comm *model.ListCommunication) *model.AppError {
	if err := a.store.List().AddDestination(domainId, search, comm); err != nil {
		return model.NewAppError("ListAddCommunication", "store.list.add_destination", nil, err.Error(), http.StatusInternalServerError)
	}
	return nil
}
