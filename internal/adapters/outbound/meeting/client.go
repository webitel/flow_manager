package meeting

import (
	"context"
	"fmt"
	"sync"

	"github.com/webitel/wlog"

	"github.com/webitel/flow_manager/api/gen/web-meeting-backend"
	"github.com/webitel/flow_manager/internal/infrastructure/grpcdial"
)

const serviceName = "web_meeting_backend"

type Client struct {
	consulAddr string
	startOnce  sync.Once
	api        *grpcdial.Client[web_meeting_backend.MeetingServiceClient]
}

func New(consulAddr string) *Client {
	return &Client{consulAddr: consulAddr}
}

func (c *Client) Start() error {
	wlog.Debug("starting " + serviceName + " client")
	var err error
	c.startOnce.Do(func() {
		c.api, err = grpcdial.NewClient(c.consulAddr, serviceName, web_meeting_backend.NewMeetingServiceClient)
	})
	return err
}

func (c *Client) Stop() {
	_ = c.api.Close()
}

func (c *Client) Create(ctx context.Context, domainId int64, title string, expireSec int, basePath string, vars map[string]string) (string, error) {
	res, err := c.api.API.CreateMeetingNA(ctx, &web_meeting_backend.CreateMeetingRequest{
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
	res, err := c.api.API.GetMeeting(ctx, &web_meeting_backend.GetMeetingRequest{Id: id})
	if err != nil {
		return nil, fmt.Errorf("meeting.Get: %w", err)
	}
	return res.Variables, nil
}
