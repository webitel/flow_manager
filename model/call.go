package model

import (
	"fmt"
	"strconv"
)

type CallResponse struct {
	Status string
}

var CallResponseOK = &CallResponse{"SUCCESS"}
var CallResponseError = &CallResponse{"ERROR"}

type CallRouter interface {
	Router
}

func (r CallResponse) String() string {
	return r.Status
}

type CallDirection string

const (
	CallDirectionInbound  CallDirection = "inbound"
	CallDirectionOutbound               = "outbound"
)

const (
	CallEndpointTypeUser        = "user"
	CallEndpointTypeGateway     = "gateway"
	CallEndpointTypeDestination = "dest"
)

type CallEndpoint struct {
	Type   string
	Id     string
	Number string
	Name   string
}

func (c *CallEndpoint) String() string {
	if c == nil {
		return "empty"
	}
	return fmt.Sprintf("type: %s number: %s name: \"%s\" id: %s", c.Type, c.Number, c.Name, c.Id)
}

func (c CallEndpoint) IntId() *int {
	if r, e := strconv.Atoi(c.Id); e != nil {
		return nil
	} else {
		return NewInt(r)
	}
}

type Call interface {
	Connection
	//ParentType() *string //TODO transfer logic
	From() *CallEndpoint
	To() *CallEndpoint

	Direction() CallDirection
	Destination() string
	DomainId() int
	SetDomainName(name string)
	DomainName() string

	SetAll(vars Variables) (Response, *AppError)
	SetNoLocal(vars Variables) (Response, *AppError)

	RingReady() (Response, *AppError)
	PreAnswer() (Response, *AppError)
	Answer() (Response, *AppError)
	Hangup(cause string) (Response, *AppError)
	HangupNoRoute() (Response, *AppError)
	HangupAppErr() (Response, *AppError)
	Bridge(call Call, strategy string, vars map[string]string, endpoints []*Endpoint) (Response, *AppError)
}
