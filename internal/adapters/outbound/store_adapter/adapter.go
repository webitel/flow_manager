// Package store_adapter wraps storage.Store and exposes the thin delegating
// methods that used to live in app/.
package store_adapter

import (
	"context"
	"fmt"
	"time"

	"github.com/webitel/flow_manager/internal/domain/call"
	"github.com/webitel/flow_manager/internal/domain/calendar"
	chatdomain "github.com/webitel/flow_manager/internal/domain/chat"
	emaildomain "github.com/webitel/flow_manager/internal/domain/email"
	"github.com/webitel/flow_manager/internal/domain/files"
	"github.com/webitel/flow_manager/internal/domain/flow"
	listdomain "github.com/webitel/flow_manager/internal/domain/list"
	"github.com/webitel/flow_manager/internal/domain/queue"
	"github.com/webitel/flow_manager/internal/domain/user"
	"github.com/webitel/flow_manager/internal/infrastructure/cache"
	"github.com/webitel/flow_manager/internal/storage"
)

// Adapter wraps a storage.Store and provides thin delegating methods.
// Embed *Adapter in FlowManager to promote all methods without re-declaring
// them one by one in app/.
type Adapter struct {
	store         storage.Store
	externalStore *cache.ExternalStoreManager // optional; set via SetExternalStore
}

// New creates a new Adapter backed by s.
func New(s storage.Store) *Adapter {
	return &Adapter{store: s}
}

// toError wraps a plain error adding context; nil in → nil out.
func toError(op string, err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s: %w", op, err)
}

// ── Media ─────────────────────────────────────────────────────────────────────

func (a *Adapter) GetMediaFiles(domainId int64, req *[]*call.PlaybackFile) ([]*call.PlaybackFile, error) {
	res, err := a.store.Media().GetFiles(domainId, req)
	return res, toError("App.GetMediaFiles", err)
}

func (a *Adapter) GetMediaFile(domainId int64, id int) (*files.File, error) {
	res, err := a.store.Media().Get(domainId, id)
	return res, toError("App.GetMediaFile", err)
}

func (a *Adapter) SearchMediaFile(domainId int64, search *files.SearchFile) (*files.File, error) {
	res, err := a.store.Media().SearchOne(domainId, search)
	return res, toError("App.SearchMediaFile", err)
}

func (a *Adapter) GetPlaybackFile(domainId int64, search *call.PlaybackFile) (*call.PlaybackFile, error) {
	res, err := a.store.Media().GetPlaybackFile(domainId, search)
	return res, toError("App.GetPlaybackFile", err)
}

// ── Log ───────────────────────────────────────────────────────────────────────

func (a *Adapter) StoreLog(schemaId int, connId string, log []*flow.StepLog) error {
	if len(log) == 0 {
		return nil
	}
	if err := a.store.Log().Save(schemaId, connId, log); err != nil {
		return fmt.Errorf("StoreLog: store.log.save: %w", err)
	}
	return nil
}

// ── Queue ─────────────────────────────────────────────────────────────────────

func (a *Adapter) FindQueueByName(domainId int64, name string) (int32, error) {
	id, err := a.store.Queue().FindQueueByName(domainId, name)
	if err != nil {
		return 0, fmt.Errorf("FindQueueByName: store.queue.find_by_name: %w", err)
	}
	return id, nil
}

// ── User ──────────────────────────────────────────────────────────────────────

func (a *Adapter) GetUserProperties(domainId int64, search *user.SearchUser, mapRes flow.Variables) (flow.Variables, error) {
	res, err := a.store.User().GetProperties(domainId, search, mapRes)
	if err != nil {
		return nil, fmt.Errorf("GetUserProperties: store.user.get_properties: %w", err)
	}
	return res, nil
}

func (a *Adapter) GetAgentIdByExtension(domainId int64, extension string) (*int32, error) {
	res, err := a.store.User().GetAgentIdByExtension(domainId, extension)
	if err != nil {
		return nil, fmt.Errorf("GetAgentIdByExtension: store.user.get_agent_id_by_extension: %w", err)
	}
	return res, nil
}

// ── Call ──────────────────────────────────────────────────────────────────────

func (a *Adapter) SetCallGranteeId(domainId int64, id string, granteeId int64) error {
	return toError("App.SetCallGranteeId", a.store.Call().SetGranteeId(domainId, id, granteeId))
}

func (a *Adapter) SetBlindTransferNumber(domainId int64, callId, destination string) error {
	return toError("App.SetBlindTransferNumber", a.store.Call().SetBlindTransfer(domainId, callId, destination))
}

func (a *Adapter) CallSetContactId(domainId int64, callId string, contactId int64) error {
	return toError("App.CallSetContactId", a.store.Call().SetContactId(domainId, callId, contactId))
}

func (a *Adapter) StoreCallVariables(id string, vars map[string]string) error {
	if len(vars) == 0 {
		return nil
	}
	cv := make(call.CallVariables)
	for k, v := range vars {
		cv[k] = v
	}
	return toError("App.StoreCallVariables", a.store.Call().SetVariables(id, &cv))
}

func (a *Adapter) UpdateCallFrom(id string, name, number, destination *string) error {
	return toError("App.UpdateCallFrom", a.store.Call().UpdateFrom(id, name, number, destination))
}

func (a *Adapter) LastBridgedCall(domainId int64, number, hours string, dialer, inbound, outbound *string, queueIds []int, mapRes flow.Variables) (flow.Variables, error) {
	res, err := a.store.Call().LastBridged(domainId, number, hours, dialer, inbound, outbound, queueIds, mapRes)
	return res, toError("App.LastBridgedCall", err)
}

func (a *Adapter) SetCallUserId(domainId int64, id string, userId int64) error {
	return toError("App.SetCallUserId", a.store.Call().SetUserId(domainId, id, userId))
}

// ── Member ────────────────────────────────────────────────────────────────────

func (a *Adapter) GetCallPosition(callId string) (int64, error) {
	res, err := a.store.Member().CallPosition(callId)
	return res, toError("App.GetCallPosition", err)
}

func (a *Adapter) GetMemberProperties(domainId int64, search *queue.SearchMember, mapRes flow.Variables) (flow.Variables, error) {
	res, err := a.store.Member().GetProperties(domainId, search, mapRes)
	return res, toError("App.GetMemberProperties", err)
}

func (a *Adapter) PatchMembers(domainId int64, search *queue.SearchMember, patch *queue.PatchMember) (int, error) {
	res, err := a.store.Member().PatchMembers(domainId, search, patch)
	return res, toError("App.PatchMembers", err)
}

func (a *Adapter) EWTPuzzle(domainId int64, callId string, min int, queueIds []int, bucketIds []int) (float64, error) {
	res, err := a.store.Member().EWTPuzzle(domainId, callId, min, queueIds, bucketIds)
	return res, toError("App.EWTPuzzle", err)
}

// ── Calendar ──────────────────────────────────────────────────────────────────

func (a *Adapter) CheckCalendar(domainId int64, id *int, name *string) (*calendar.Calendar, error) {
	c, err := a.store.Calendar().Check(domainId, id, name)
	if err != nil {
		return nil, fmt.Errorf("CheckCalendar: store.calendar.check: %w", err)
	}
	return c, nil
}

// ── Email ─────────────────────────────────────────────────────────────────────

func (a *Adapter) GetEmailProperties(domainId int64, id *int64, messageId *string, mapRes flow.Variables) (flow.Variables, error) {
	vars, err := a.store.Email().GerProperties(domainId, id, messageId, mapRes)
	if err != nil {
		return nil, fmt.Errorf("GetEmailProperties: store.email.get_properties: %w", err)
	}
	return vars, nil
}

func (a *Adapter) MailSetContacts(ctx context.Context, domainId int64, id string, contactIds []int64) error {
	if err := a.store.Email().SetContact(ctx, domainId, id, contactIds); err != nil {
		return fmt.Errorf("MailSetContacts: store.email.set_contact: %w", err)
	}
	return nil
}

func (a *Adapter) SaveEmail(domainId int64, email *emaildomain.Email) error {
	return a.store.Email().Save(domainId, email)
}

func (a *Adapter) GetFileMetadata(domainId int64, ids []int64) ([]files.File, error) {
	return a.store.File().GetMetadata(domainId, ids)
}

// ── Chat (store-only) ─────────────────────────────────────────────────────────

func (a *Adapter) LastBridgedChat(domainId int64, number, hours string, queueIds []int, mapRes flow.Variables) (flow.Variables, error) {
	vars, err := a.store.Chat().LastBridged(domainId, number, hours, queueIds, mapRes)
	if err != nil {
		return nil, fmt.Errorf("LastBridgedChat: store.chat.last_bridged: %w", err)
	}
	return vars, nil
}

func (a *Adapter) ChatProfileType(domainId int64, profileId int) (string, error) {
	return a.store.Chat().ProfileType(domainId, profileId)
}

func (a *Adapter) GetChatMessagesByConversationId(ctx context.Context, domainId int64, conversationId string, limit int64) (*[]chatdomain.ChatMessage, error) {
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
	comm := &listdomain.ListCommunication{
		Destination: destination,
		Description: description,
	}
	if expireAtMS > 0 {
		t := time.UnixMilli(expireAtMS)
		comm.ExpireAt = &t
	}
	return a.ListAddCommunication(domainId, &queue.SearchEntity{Id: listId, Name: listName}, comm)
}

func (a *Adapter) ListCheckNumber(domainId int64, number string, listId *int, listName *string) (bool, error) {
	ok, err := a.store.List().CheckNumber(domainId, number, listId, listName)
	if err != nil {
		return false, fmt.Errorf("ListCheckNumber: store.list.check_number: %w", err)
	}
	return ok, nil
}

func (a *Adapter) ListAddCommunication(domainId int64, search *queue.SearchEntity, comm *listdomain.ListCommunication) error {
	if err := a.store.List().AddDestination(domainId, search, comm); err != nil {
		return fmt.Errorf("ListAddCommunication: store.list.add_destination: %w", err)
	}
	return nil
}
