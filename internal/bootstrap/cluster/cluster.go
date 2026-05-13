// Package cluster handles service discovery registration and deregistration.
package cluster

import (
	"fmt"

	bscfg "github.com/webitel/flow_manager/internal/bootstrap/config"
	"github.com/webitel/flow_manager/internal/infrastructure/discovery"
)

// Cluster manages Consul service registration for this node.
type Cluster struct {
	connection   string
	id           string
	discoveryURL string
	host         string
	port         int
	Discovery    discovery.ServiceDiscovery
}

// New creates a Cluster for the given node id and Consul URL.
// host/port are the gRPC advertise coordinates (may be empty/zero when no
// gRPC server is configured).
func New(id, discoveryURL, host string, port int) *Cluster {
	return &Cluster{
		id:           id,
		discoveryURL: discoveryURL,
		host:         host,
		port:         port,
	}
}

func (c *Cluster) Start() error {
	sd, err := discovery.NewServiceDiscovery(c.id, c.discoveryURL, func() (b bool, appError error) {
		return true, nil
	})
	if err != nil {
		return err
	}

	c.Discovery = sd

	if err = sd.RegisterService(bscfg.AppServiceName, c.host, c.port, bscfg.AppServiceTTL, bscfg.AppDeregisterCriticalTTL); err != nil {
		return err
	}

	c.connection = fmt.Sprintf("%s-%s", bscfg.AppServiceName, c.id)

	return nil
}

func (c *Cluster) Stop() {
	if c.Discovery != nil {
		c.Discovery.Shutdown()
	}
}

func (c *Cluster) ConnectionString() string {
	return c.connection
}
