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

type QueueInfo struct {
	QueueId   int    `json:"queue_id,string"`
	AttemptId int64  `json:"attempt_id,string"`
	TeamId    *int   `json:"team_id,string"`
	AgentId   *int   `json:"agent_id,string"`
	MemberId  *int64 `json:"member_id,string"`
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
	Queue       *QueueInfo     `json:"queue"`
}

type CallActionRinging struct {
	CallAction
	CallActionInfo
}

func (r *CallActionRinging) GetQueueId() *int {
	if r.Queue != nil {
		return &r.Queue.QueueId
	}
	return nil
}

func (r *CallActionRinging) GetAttemptId() *int64 {
	if r.Queue != nil {
		return &r.Queue.AttemptId
	}
	return nil
}

func (r *CallActionRinging) GetTeamId() *int {
	if r.Queue != nil {
		return r.Queue.TeamId
	}
	return nil
}

func (r *CallActionRinging) GetAgentId() *int {
	if r.Queue != nil {
		return r.Queue.AgentId
	}
	return nil
}

func (r *CallActionRinging) GetMemberIdId() *int64 {
	if r.Queue != nil {
		return r.Queue.MemberId
	}
	return nil
}

func (r *CallActionRinging) GetFrom() *CallEndpoint {
	if r != nil {
		return r.From
	}
	return nil
}

func (r *CallActionRinging) GetTo() *CallEndpoint {
	if r != nil {
		return r.To
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
	SetDomainName(name string)
	DomainName() string

	SetAll(vars Variables) (Response, *AppError)
	SetNoLocal(vars Variables) (Response, *AppError)
	UnSet(name string) (Response, *AppError)

	RingReady() (Response, *AppError)
	PreAnswer() (Response, *AppError)
	Answer() (Response, *AppError)
	Echo(delay int) (Response, *AppError)
	Hangup(cause string) (Response, *AppError)
	HangupNoRoute() (Response, *AppError)
	HangupAppErr() (Response, *AppError)
	Bridge(call Call, strategy string, vars map[string]string, endpoints []*Endpoint, codec []string) (Response, *AppError)
	Sleep(int) (Response, *AppError)
	Conference(name, profile, pin string, tags []string) (Response, *AppError)
	RecordFile(name, format string, maxSec, silenceThresh, silenceHits int) (Response, *AppError)
	RecordSession(name, format string, minSec int, stereo, bridged, followTransfer bool) (Response, *AppError)
	RecordSessionStop(name, format string) (Response, *AppError)
	Export(vars []string) (Response, *AppError)
	FlushDTMF() (Response, *AppError)
	StartDTMF() (Response, *AppError)
	StopDTMF() (Response, *AppError)
	Park(name string, in bool, lotFrom, lotTo string) (Response, *AppError)
	Playback(files []*PlaybackFile) (Response, *AppError)
	PlaybackAndGetDigits(files []*PlaybackFile, params *PlaybackDigits) (Response, *AppError)
	Redirect(uri []string) (Response, *AppError)
	SetSounds(lang, voice string) (Response, *AppError)
	ScheduleHangup(sec int, cause string) (Response, *AppError)
}

type PlaybackFile struct {
	Type *string `json:"type"`
	Id   *int    `json:"id"`
	Name *string `json:"name"`
}

type PlaybackDigits struct {
	SetVar    *string `json:"setVar"`
	Min       *int    `json:"min" def:"1"`
	Max       *int    `json:"max"`
	Tries     *int    `json:"tries"`
	Timeout   *int    `json:"timeout"`
	FlushDtmf bool    `json:"flushDTMF"`
	Regexp    *string `json:"regexp"`
}

type PlaybackArgs struct {
	Files      []*PlaybackFile `json:"files"`
	Terminator string          `json:"terminator" def:"#"`
	GetDigits  *PlaybackDigits `json:"getDigits"`
}
