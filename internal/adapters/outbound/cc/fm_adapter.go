// Package cc — FMAdapter wraps CCManager and exposes the thin delegating methods
// that used to live in app/cc.go.
package cc

import (
	"context"
	"fmt"
	"net/http"

	cc2 "github.com/webitel/flow_manager/api/gen/cc"
	"github.com/webitel/flow_manager/api/gen/engine"
	"github.com/webitel/flow_manager/internal/domain/call"
	domcc "github.com/webitel/flow_manager/internal/domain/cc"
	"github.com/webitel/flow_manager/internal/domain/queue"
	apperrs "github.com/webitel/flow_manager/internal/infrastructure/errors"
	"github.com/webitel/flow_manager/internal/storage"
)

// FMAdapter wraps a CCManager and a storage.Store, exposing the higher-level
// helper methods that used to live in app/cc.go.
// Embed *FMAdapter in RouterDeps to promote all methods.
type FMAdapter struct {
	cc    domcc.CCManager
	store storage.Store
}

// NewFMAdapter creates a new FMAdapter.
func NewFMAdapter(cc domcc.CCManager, st storage.Store) *FMAdapter {
	return &FMAdapter{cc: cc, store: st}
}

func (a *FMAdapter) JoinToInboundQueue(ctx context.Context, in *cc2.CallJoinToQueueRequest) (cc2.MemberService_CallJoinToQueueClient, error) {
	return a.cc.Member().JoinCallToQueue(ctx, in)
}

func (a *FMAdapter) CallOutboundQueue(ctx context.Context, in *cc2.OutboundCallRequest) (*cc2.OutboundCallResponse, error) {
	return a.cc.Member().CallOutbound(ctx, in)
}

func (a *FMAdapter) JoinChatToInboundQueue(ctx context.Context, in *cc2.ChatJoinToQueueRequest) (cc2.MemberService_ChatJoinToQueueClient, error) {
	return a.cc.Member().JoinChatToQueue(ctx, in)
}

func (a *FMAdapter) CreateMember(domainId int64, queueId, holdSec int, member *queue.CallbackMember) error {
	if err := a.store.Member().CreateMember(domainId, queueId, holdSec, member); err != nil {
		return fmt.Errorf("App.CreateMember: app.store_err: %w", err)
	}
	return nil
}

func (a *FMAdapter) JoinToAgent(ctx context.Context, in *cc2.CallJoinToAgentRequest) (cc2.MemberService_CallJoinToAgentClient, error) {
	return a.cc.Member().CallJoinToAgent(ctx, in)
}

func (a *FMAdapter) TaskJoinToAgent(ctx context.Context, in *cc2.TaskJoinToAgentRequest) (cc2.MemberService_TaskJoinToAgentClient, error) {
	return a.cc.Member().TaskJoinToAgent(ctx, in)
}

func (a *FMAdapter) CancelUserDistribute(ctx context.Context, domainId int64, extension string) error {
	agentId, storeErr := a.store.User().GetAgentIdByExtension(domainId, extension)
	if storeErr != nil {
		return fmt.Errorf("CancelUserDistribute: store.user.get_agent_id: %w", storeErr)
	}

	if agentId == nil {
		return nil
	}

	_, perr := a.cc.Member().CancelAgentDistribute(ctx, &cc2.CancelAgentDistributeRequest{
		AgentId: *agentId,
	})
	if perr != nil {
		return apperrs.Newf(http.StatusNotFound, "App.CancelUserDistribute: %s", perr.Error())
	}

	return nil
}

func (a *FMAdapter) AttemptResult(result *call.AttemptResult) error {
	req := &cc2.AttemptResultRequest{
		AttemptId:                   result.Id,
		Status:                      result.Status,
		Variables:                   result.Variables,
		Display:                     result.StickyDisplay,
		Description:                 result.Description,
		AgentId:                     result.AgentId,
		Redial:                      result.Redial,
		ExcludeCurrentCommunication: result.ExcludeCurrentCommunication,
	}

	if result.ExpiredAt != nil {
		req.ExpireAt = *result.ExpiredAt
	}
	if result.ReadyAt != nil {
		req.NextDistributeAt = *result.ReadyAt
	}
	if result.WaitBetweenRetries != nil {
		req.WaitBetweenRetries = *result.WaitBetweenRetries
	}
	req.AddCommunications = ccCommunications(result.AddCommunications)

	if err := a.cc.Member().AttemptResult(req); err != nil {
		return fmt.Errorf("AttemptResult: app.attempt.result: %w", err)
	}
	return nil
}

func (a *FMAdapter) JoinIMToInboundQueue(ctx context.Context, in *cc2.IMJoinToQueueRequest) (int64, <-chan domcc.QueueEvent, error) {
	res, err := a.cc.Member().JoinIMToQueue(ctx, in)
	if err != nil {
		return 0, nil, err
	}
	ch := a.cc.SubscribeAttempt(res.AttemptId)
	return res.AttemptId, ch, nil
}

func (a *FMAdapter) LeavingIMToInboundQueue(attId int64) {
	a.cc.UnSubscribeAttempt(attId)
}

func (a *FMAdapter) CancelAttempt(ctx context.Context, att queue.InQueueKey, result string) error {
	if err := a.cc.Member().CancelAttempt(ctx, att.AttemptId, result, att.AppId); err != nil {
		return fmt.Errorf("CancelAttempt: app.attempt.cancel: %w", err)
	}
	return nil
}

func (a *FMAdapter) ResumeAttempt(ctx context.Context, attemptId, domainId int64) error {
	return a.cc.Member().ResumeAttempt(ctx, attemptId, domainId)
}

func ccCommunications(r []queue.CallbackCommunication) []*cc2.MemberCommunicationCreateRequest {
	l := len(r)
	if l == 0 {
		return nil
	}

	comm := make([]*cc2.MemberCommunicationCreateRequest, 0, l)
	for _, v := range r {
		if v.Destination != "" && v.Type.Id != nil {
			c := &cc2.MemberCommunicationCreateRequest{
				Destination: v.Destination,
				Type: &engine.Lookup{
					Id: int64(*v.Type.Id),
				},
			}
			if v.Priority != nil {
				c.Priority = int32(*v.Priority)
			}
			if v.Description != nil {
				c.Description = *v.Description
			}
			if v.ResourceId != nil {
				c.Resource = &engine.Lookup{
					Id: int64(*v.ResourceId),
				}
			}
			if v.Display != nil {
				c.Display = *v.Display
			}
			comm = append(comm, c)
		}
	}
	return comm
}
