package consul

import "github.com/hashicorp/consul/api"

//go:generate mockery --name Agent --output mocks --outpkg mocks --case underscore
type Agent interface {
	ServiceRegister(service *api.AgentServiceRegistration) error
	ServiceDeregister(serviceID string) error
	PassTTL(checkID, note string) error
	FailTTL(checkID, note string) error
}
