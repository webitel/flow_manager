package model

import (
	"encoding/json"
	"fmt"
	"github.com/webitel/wlog"
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
	CallExchange       = "call"
	CallEventQueueName = "workflow-call"
)

const (
	CallDirectionInbound  CallDirection = "inbound"
	CallDirectionOutbound               = "outbound"
)

const (
	CallEndpointTypeUser        = "user"
	CallEndpointTypeGateway     = "gateway"
	CallEndpointTypeDestination = "dest"
)

const (
	CallActionRingingName = "ringing"
	CallActionActiveName  = "active"
	CallActionBridgeName  = "bridge"
	CallActionHoldName    = "hold"
	CallActionDtmfName    = "dtmf"
	CallActionHangupName  = "hangup"
)

type CallAction struct {
	Id        string `json:"id"`
	AppId     string `json:"app_id"`
	DomainId  int8   `json:"domain_id,string"`
	Timestamp int64  `json:"timestamp,string"`
	Event     string `json:"event"`
}

type CallActionData struct {
	CallAction
	Data   *string     `json:"data,omitempty"`
	parsed interface{} `json:"-"`
}

type CallEndpoint struct {
	Type   string
	Id     string
	Number string
	Name   string
}

func (e *CallEndpoint) GetType() *string {
	if e != nil {
		return &e.Type
	}

	return nil
}

func (e *CallEndpoint) GetId() *string {
	if e != nil {
		return &e.Id
	}

	return nil
}

func (e *CallEndpoint) GetNumber() *string {
	if e != nil {
		return &e.Number
	}

	return nil
}

func (e *CallEndpoint) GetName() *string {
	if e != nil {
		return &e.Name
	}

	return nil
}

type CallActionInfo struct {
	GatewayId   *int           `json:"gateway_id"`
	UserId      *int           `json:"user_id"`
	Direction   string         `json:"direction"`
	Destination string         `json:"destination"`
	From        *CallEndpoint  `json:"from"`
	To          *CallEndpoint  `json:"to"`
	ParentId    *string        `json:"parent_id"`
	Payload     *CallVariables `json:"payload"`
}

type CallActionRinging struct {
	CallAction
	CallActionInfo
}

func (c *CallActionRinging) GetFrom() *CallEndpoint {
	if c != nil {
		return c.From
	}
	return nil
}

func (c *CallActionRinging) GetTo() *CallEndpoint {
	if c != nil {
		return c.To
	}
	return nil
}

type CallActionActive struct {
	CallAction
}

type CallActionHold struct {
	CallAction
}

type CallActionBridge struct {
	CallAction
	BridgedId string `json:"bridged_id"`
}

type CallActionHangup struct {
	CallAction
	Cause         string `json:"cause"`
	SipCode       *int   `json:"sip"`
	OriginSuccess *bool  `json:"originate_success"`
}

type CallVariables map[string]interface{}

func (c *CallActionData) GetEvent() interface{} {
	if c.parsed != nil {
		return c.parsed
	}

	switch c.Event {
	case CallActionRingingName:
		c.parsed = &CallActionRinging{
			CallAction: c.CallAction,
		}
	case CallActionActiveName:
		c.parsed = &CallActionActive{
			CallAction: c.CallAction,
		}

	case CallActionHoldName:
		c.parsed = &CallActionHold{
			CallAction: c.CallAction,
		}

	case CallActionBridgeName:
		c.parsed = &CallActionBridge{
			CallAction: c.CallAction,
		}
	case CallActionHangupName:
		c.parsed = &CallActionHangup{
			CallAction: c.CallAction,
		}
	}

	if c.Data != nil {
		if err := json.Unmarshal([]byte(*c.Data), &c.parsed); err != nil {
			wlog.Error(fmt.Sprintf("parse call %s [%s] error: %s", c.Id, c.Event, err.Error()))
		}
	}
	return c.parsed
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
	Echo(delay int) (Response, *AppError)
	Hangup(cause string) (Response, *AppError)
	HangupNoRoute() (Response, *AppError)
	HangupAppErr() (Response, *AppError)
	Bridge(call Call, strategy string, vars map[string]string, endpoints []*Endpoint) (Response, *AppError)
	Sleep(int) (Response, *AppError)
}
