package app

import (
	"github.com/webitel/engine/discovery"
	"github.com/webitel/flow_manager/model"
)

type cluster struct {
	app       *FlowManager
	discovery discovery.ServiceDiscovery
}

func NewCluster(app *FlowManager) *cluster {
	return &cluster{
		app: app,
	}
}

func (c *cluster) Start() error {
	sd, err := discovery.NewServiceDiscovery(c.app.id, c.app.Config().DiscoverySettings.Url, func() (b bool, appError error) {
		return true, nil
	})
	if err != nil {
		return err
	}
	c.discovery = sd

	host, port := c.app.GetPublicInterface()
	err = sd.RegisterService(model.AppServiceName, host, port, model.AppServiceTTL, model.AppDeregesterCriticalTTL)
	if err != nil {
		return err
	}

	return nil
}

func (f *FlowManager) GetPublicInterface() (string, int) {

	return f.servers[0].Host(), f.servers[0].Port()
}

func (c *cluster) Stop() {
	c.discovery.Shutdown()
}
