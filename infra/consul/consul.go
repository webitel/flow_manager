package consul

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/hashicorp/consul/api"

	"github.com/webitel/wlog"
)

type CheckFunction func() error

type Consul struct {
	id                string
	agent             Agent
	stop              chan struct{}
	check             CheckFunction
	checkID           string
	ready             bool
	config            *Config
	log               *wlog.Logger
	serviceInstanceID string
}

type Config struct {
	Name            string
	Address         string
	Port            int
	TTL             time.Duration
	CriticalTTL     time.Duration
	Tags            []string
	ConsulAgentAddr string
}

// NewConsul створює новий екземпляр Consul клієнта.
// id: унікальний ідентифікатор для екземпляра сервісу.
// check: функція, яка повертає nil, якщо сервіс здоровий, або error, якщо ні.
// log: logger
// consulAgentAddr: адреса Consul агента.
func NewConsul(id, consulAgentAddr string, log *wlog.Logger, check CheckFunction) (*Consul, error) {
	if check == nil {
		return nil, errors.New("check function cannot be nil")
	}

	conf := api.DefaultConfig()
	conf.Address = consulAgentAddr

	cli, err := api.NewClient(conf)
	if err != nil {
		return nil, fmt.Errorf("failed to create Consul client: %w", err)
	}

	c := &Consul{
		id:      id,
		log:     log,
		agent:   cli.Agent(),
		stop:    make(chan struct{}),
		check:   check,
		checkID: fmt.Sprintf("service:%s", id), // CheckID завжди має префікс "service:"
	}

	return c, nil
}

// RegisterService реєструє сервіс в Consul.
// config: конфігурація сервісу для реєстрації.
func (c *Consul) RegisterService(config Config) error {
	c.config = &config

	c.serviceInstanceID = fmt.Sprintf("%s-%s", config.Name, c.id)

	serviceRegistration := &api.AgentServiceRegistration{
		ID:      c.serviceInstanceID,
		Name:    config.Name,
		Tags:    config.Tags,
		Address: config.Address,
		Port:    config.Port,
		Check: &api.AgentServiceCheck{
			DeregisterCriticalServiceAfter: config.CriticalTTL.String(),
			TTL:                            config.TTL.String(),
			CheckID:                        c.checkID,
		},
	}

	if err := c.agent.ServiceRegister(serviceRegistration); err != nil {
		return fmt.Errorf("failed to register service %s in Consul: %w", serviceRegistration.Name, err)
	}

	c.log.Info(fmt.Sprintf("Service '%s' (ID: %s) registered with Consul.", serviceRegistration.Name, serviceRegistration.ID))

	go c.startTTLUpdater(config.TTL / 2)

	c.updateTTLStatus()

	return nil
}

func (c *Consul) startTTLUpdater(interval time.Duration) {
	// --- FIX STARTS HERE ---
	// Add a guard clause to prevent panics from a zero or negative interval.
	if interval <= 0 {
		c.log.Error(fmt.Sprintf("Invalid TTL interval (%v) for service ID: %s. TTL updater will not start.", interval, c.serviceInstanceID))

		return
	}
	// --- FIX ENDS HERE ---

	defer c.log.Info(fmt.Sprintf("Stopped Consul TTL updater for service ID: %s", c.serviceInstanceID))

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-c.stop:
			return
		case <-ticker.C:
			c.updateTTLStatus()
		}
	}
}

func (c *Consul) updateTTLStatus() {
	err := c.check()
	if err != nil {
		if agentErr := c.agent.FailTTL(c.checkID, err.Error()); agentErr != nil {
			c.handleTTLUpdateError(agentErr)
		}

		c.ready = false
	} else {
		if agentErr := c.agent.PassTTL(c.checkID, "Service is healthy."); agentErr != nil {
			c.handleTTLUpdateError(agentErr)
		} else {
			c.ready = true
		}
	}
}

func (c *Consul) handleTTLUpdateError(err error) {
	var apiErr api.StatusError
	if errors.As(err, &apiErr) {
		// Перевіряємо, чи це помилка сервера Consul
		if apiErr.Code == http.StatusInternalServerError {
			c.log.Error(fmt.Sprintf("Consul returned internal server error during TTL update. Attempting to re-register service ID: %s. Error: %s", c.id, err.Error()))

			if c.config != nil {
				if regErr := c.RegisterService(*c.config); regErr != nil {
					c.log.Error(fmt.Sprintf("Failed to re-register service %s (ID: %s) with Consul: %s", c.config.Name, c.id, regErr.Error()))
				}
			} else {
				c.log.Error(fmt.Sprintf("Consul config is nil, cannot re-register service ID: %s", c.id))
			}
		}
	} else {
		c.log.Error(fmt.Sprintf("Error updating Consul TTL for service ID: %s. %s", c.id, err.Error()))
	}
}

func (c *Consul) IsReady() bool {
	return c.ready
}

func (c *Consul) Shutdown() {
	c.log.Info(fmt.Sprintf("Deregistering service ID: %s from Consul...", c.id))
	close(c.stop) // Сигналізуємо горутині зупинитися

	if err := c.agent.ServiceDeregister(c.serviceInstanceID); err != nil {
		c.log.Error(fmt.Sprintf("Failed to deregister service ID: %s from Consul: %s", c.id, err.Error()))
	} else {
		c.log.Info(fmt.Sprintf("Service ID: %s successfully deregistered from Consul.", c.id))
	}
}
