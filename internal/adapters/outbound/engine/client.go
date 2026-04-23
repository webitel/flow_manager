package engine

import (
	"sync"

	"github.com/webitel/wlog"

	"github.com/webitel/flow_manager/gen/engine"
	"github.com/webitel/flow_manager/infra/grpcdial"
)

const (
	serviceName = "engine"
)

type Client struct {
	consulAddr string
	startOnce  sync.Once
	call       *grpcdial.Client[engine.CallServiceClient]
	feedback   *grpcdial.Client[engine.FeedbackServiceClient]
}

func New(consulAddr string) *Client {
	cli := &Client{
		consulAddr: consulAddr,
	}

	return cli
}

func (cm *Client) Start() error {
	wlog.Debug("starting engine client")
	var err error
	cm.startOnce.Do(func() {
		cm.call, err = grpcdial.NewClient(cm.consulAddr, serviceName, engine.NewCallServiceClient)
		if err != nil {
			return
		}
		cm.feedback, err = grpcdial.NewClient(cm.consulAddr, serviceName, engine.NewFeedbackServiceClient)
		if err != nil {
			return
		}
	})
	return err
}

func (cm *Client) Call() engine.CallServiceClient {
	return cm.call.API
}

func (cm *Client) Feedback() engine.FeedbackServiceClient {
	return cm.feedback.API
}

func (cm *Client) Stop() {
}
