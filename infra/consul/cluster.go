package consul

import (
	"fmt"
	"time"

	"github.com/webitel/wlog"
)

var (
	defaultReconnectAttempts = 10               // Кількість спроб перепідключення
	reconnectDuration        = 5 * time.Second  // Час очікування між спробами перепідключення
	serviceTTL               = 10 * time.Second // TTL для перевірки здоров'я
	deregisterTTL            = 2 * serviceTTL   // Час до дерегістрації після критичного стану
)

var newConsul = NewConsul

type Cluster struct {
	consulAddr string
	name       string
	discovery  *Consul
	log        *wlog.Logger
}

func NewCluster(name, consulAddr string, log *wlog.Logger) *Cluster {
	return &Cluster{
		name:       name,
		consulAddr: consulAddr,
		log:        log.With(wlog.String("scope", "consul")),
	}
}

func (c *Cluster) Start(serviceInstanceID, host string, port int) error {
	consulClient, err := newConsul(
		serviceInstanceID,
		c.consulAddr,
		c.log,
		func() error {
			return nil
		},
	)
	if err != nil {
		return err
	}

	c.discovery = consulClient

	serviceConfig := Config{
		Name:            c.name,
		Address:         host,
		Port:            port,
		TTL:             serviceTTL,
		CriticalTTL:     deregisterTTL,
		Tags:            nil,
		ConsulAgentAddr: c.consulAddr,
	}
	if err = c.attemptConsulRegistration(serviceConfig); err != nil {
		return fmt.Errorf("failed to register service in Consul after multiple attempts: %w", err)
	}

	c.log.Info(fmt.Sprintf("Service '%s' (ID: %s) successfully registered with Consul.", c.name, serviceInstanceID))

	return nil
}

func (c *Cluster) Stop() {
	c.discovery.Shutdown()
}

func (c *Cluster) attemptConsulRegistration(config Config) error {
	for i := range defaultReconnectAttempts {
		err := c.discovery.RegisterService(config)
		if err == nil {
			return nil // Успішна реєстрація
		}

		c.log.Error(fmt.Sprintf("Attempt %d/%d: Failed to register service '%s' with Consul. Retrying in %v. Error: %v",
			i+1, defaultReconnectAttempts, config.Name, reconnectDuration, err))

		time.Sleep(reconnectDuration)
	}

	return fmt.Errorf("exceeded maximum reconnect attempts (%d) for Consul registration", defaultReconnectAttempts)
}
