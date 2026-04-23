package cc

import (
	"context"

	genpb "github.com/webitel/flow_manager/gen/cc"
	"github.com/webitel/flow_manager/infra/grpcdial"
	domcc "github.com/webitel/flow_manager/internal/domain/cc"
)

type agentApi struct {
	*grpcdial.Client[genpb.AgentServiceClient]
}

func NewAgentApi(c *grpcdial.Client[genpb.AgentServiceClient]) domcc.AgentApi {
	return &agentApi{
		Client: c,
	}
}

func (api *agentApi) Online(domainId, agentId int64, onDemand bool) error {
	_, err := api.API.Online(context.TODO(), &genpb.OnlineRequest{
		AgentId:  agentId,
		OnDemand: onDemand,
		DomainId: domainId,
	})
	return err
}

func (api *agentApi) Offline(domainId, agentId int64) error {
	_, err := api.API.Offline(context.TODO(), &genpb.OfflineRequest{
		AgentId:  agentId,
		DomainId: domainId,
	})
	return err
}

func (api *agentApi) Pause(domainId, agentId int64, payload string, timeout int) error {
	_, err := api.API.Pause(context.TODO(), &genpb.PauseRequest{
		AgentId:  agentId,
		Payload:  payload,
		Timeout:  int32(timeout),
		DomainId: domainId,
	})
	return err
}

func (api *agentApi) WaitingChannel(agentId int, channel string) (int64, error) {
	if res, err := api.API.WaitingChannel(context.TODO(), &genpb.WaitingChannelRequest{
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

	_, err := api.API.AcceptTask(ctx, &genpb.AcceptTaskRequest{
		Id:       attemptId,
		AppId:    appId,
		DomainId: domainId,
	})

	return err
}

func (api *agentApi) CloseTask(appId string, domainId, attemptId int64) error {
	ctx := api.StaticHost(context.Background(), appId)

	_, err := api.API.CloseTask(ctx, &genpb.CloseTaskRequest{
		Id:       attemptId,
		AppId:    appId,
		DomainId: domainId,
	})

	return err
}

func (api *agentApi) RunTrigger(ctx context.Context, domainId, userId int64, triggerId int32, vars map[string]string) (string, error) {
	res, err := api.API.RunTrigger(ctx, &genpb.RunTriggerRequest{
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
