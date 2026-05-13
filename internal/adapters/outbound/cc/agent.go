package cc

import (
	"context"

	cc2 "github.com/webitel/flow_manager/api/gen/cc"
	domcc "github.com/webitel/flow_manager/internal/domain/cc"
	"github.com/webitel/flow_manager/internal/infrastructure/grpcdial"
)

type agentApi struct {
	*grpcdial.Client[cc2.AgentServiceClient]
}

func NewAgentApi(c *grpcdial.Client[cc2.AgentServiceClient]) domcc.AgentApi {
	return &agentApi{
		Client: c,
	}
}

func (api *agentApi) Online(domainId, agentId int64, onDemand bool) error {
	_, err := api.API.Online(context.TODO(), &cc2.OnlineRequest{
		AgentId:  agentId,
		OnDemand: onDemand,
		DomainId: domainId,
	})
	return err
}

func (api *agentApi) Offline(domainId, agentId int64) error {
	_, err := api.API.Offline(context.TODO(), &cc2.OfflineRequest{
		AgentId:  agentId,
		DomainId: domainId,
	})
	return err
}

func (api *agentApi) Pause(domainId, agentId int64, payload string, timeout int) error {
	_, err := api.API.Pause(context.TODO(), &cc2.PauseRequest{
		AgentId:  agentId,
		Payload:  payload,
		Timeout:  int32(timeout),
		DomainId: domainId,
	})
	return err
}

func (api *agentApi) WaitingChannel(agentId int, channel string) (int64, error) {
	if res, err := api.API.WaitingChannel(context.TODO(), &cc2.WaitingChannelRequest{
		AgentId: int32(agentId),
		Channel: channel,
	}); err != nil {
		return 0, err
	} else {
		return res.Timestamp, nil
	}
}

func (api *agentApi) AcceptTask(appId string, domainId, attemptId int64) error {
	ctx := api.StaticHost(context.Background(), appId)

	_, err := api.API.AcceptTask(ctx, &cc2.AcceptTaskRequest{
		Id:       attemptId,
		AppId:    appId,
		DomainId: domainId,
	})

	return err
}

func (api *agentApi) CloseTask(appId string, domainId, attemptId int64) error {
	ctx := api.StaticHost(context.Background(), appId)

	_, err := api.API.CloseTask(ctx, &cc2.CloseTaskRequest{
		Id:       attemptId,
		AppId:    appId,
		DomainId: domainId,
	})

	return err
}

func (api *agentApi) RunTrigger(ctx context.Context, domainId, userId int64, triggerId int32, vars map[string]string) (string, error) {
	res, err := api.API.RunTrigger(ctx, &cc2.RunTriggerRequest{
		DomainId:  domainId,
		TriggerId: triggerId,
		UserId:    userId,
		Variables: vars,
	})
	if err != nil {
		return "", err
	}

	return res.JobId, nil
}
