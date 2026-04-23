package meeting

import (
	"context"
	"fmt"
	"sync"

	"github.com/webitel/wlog"

	wmb "github.com/webitel/flow_manager/gen/web-meeting-backend"
	"github.com/webitel/flow_manager/infra/grpcdial"
)

const serviceName = "web_meeting_backend"

type Client struct {
	consulAddr string
	startOnce  sync.Once
	api        *grpcdial.Client[wmb.MeetingServiceClient]
}

func New(consulAddr string) *Client {
	return &Client{consulAddr: consulAddr}
}

func (c *Client) Start() error {
	wlog.Debug("starting " + serviceName + " client")
	var err error
	c.startOnce.Do(func() {
		c.api, err = grpcdial.NewClient(c.consulAddr, serviceName, wmb.NewMeetingServiceClient)
	})
	return err
}

func (c *Client) Stop() {
	_ = c.api.Close()
}

func (c *Client) Create(ctx context.Context, domainId int64, title string, expireSec int, basePath string, vars map[string]string) (string, error) {
	res, err := c.api.API.CreateMeetingNA(ctx, &wmb.CreateMeetingRequest{
		Title:     title,
		ExpireSec: int64(expireSec),
		BasePath:  basePath,
		Variables: vars,
		DomainId:  domainId,
	})
	if err != nil {
		return "", fmt.Errorf("meeting.Create: %w", err)
	}
	return res.Url, nil
}

func (c *Client) Get(ctx context.Context, id string) (map[string]string, error) {
	res, err := c.api.API.GetMeeting(ctx, &wmb.GetMeetingRequest{Id: id})
	if err != nil {
		return nil, fmt.Errorf("meeting.Get: %w", err)
	}
	return res.Variables, nil
}
