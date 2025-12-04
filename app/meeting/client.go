package meeting

import (
	"context"
	"github.com/webitel/engine/pkg/wbt"
	wmb "github.com/webitel/flow_manager/gen/web-meeting-backend"
	"github.com/webitel/flow_manager/model"
	"github.com/webitel/wlog"
	"sync"
)

const ServiceName = "web-meeting-backend"

type Client struct {
	consulAddr string
	startOnce  sync.Once
	api        *wbt.Client[wmb.MeetingServiceClient]
}

func New(consulAddr string) *Client {
	cli := &Client{
		consulAddr: consulAddr,
	}

	return cli
}

func (cm *Client) Start() error {
	wlog.Debug("starting " + ServiceName + " client")
	var err error
	cm.startOnce.Do(func() {
		cm.api, err = wbt.NewClient(cm.consulAddr, ServiceName, wmb.NewMeetingServiceClient)
		if err != nil {
			return
		}
	})
	return err
}

func (cm *Client) Stop() {
	_ = cm.api.Close()
}

func (cm *Client) CreateMeeting(ctx context.Context, title string, expireSec int, basePath string, vars map[string]string) (string, *model.AppError) {
	if cm == nil {
		return "", model.NewInternalError("meeting.client", "client is nil")
	}

	res, err := cm.api.Api.CreateMeeting(ctx, &wmb.CreateMeetingRequest{
		Title:     title,
		ExpireSec: int64(expireSec),
		BasePath:  basePath,
		Variables: vars,
	})

	if err != nil {
		return "", model.NewInternalError("meeting.CreateMeeting", err.Error())
	}

	return res.Url, nil
}

func (cm *Client) GetMeeting(ctx context.Context, id string) (map[string]string, *model.AppError) {
	if cm == nil {
		return nil, model.NewInternalError("meeting.client", "client is nil")
	}

	res, err := cm.api.Api.GetMeeting(ctx, &wmb.GetMeetingRequest{
		Id: id,
	})

	if err != nil {
		return nil, model.NewInternalError("meeting.GetMeeting", err.Error())
	}

	return res.Variables, nil
}
