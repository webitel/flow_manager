package model

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/webitel/wlog"
)

type CallResponse struct {
	Status string
}

var CallResponseOK = &CallResponse{"SUCCESS"}
var CallResponseError = &CallResponse{"ERROR"}

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
	CallActionSTTName     = "stt"
	CallActionHangupName  = "hangup"
)

type CallAction struct {
	Id        string `json:"id"`
	AppId     string `json:"app_id"`
	DomainId  int64  `json:"domain_id,string"`
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
	QueueId   *int   `json:"queue_id,string"`
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
	GranteeId   *int           `json:"grantee_id"`
}

type CallActionRinging struct {
	CallAction
	CallActionInfo
}

func (r *CallActionRinging) GetQueueId() *int {
	if r.Queue != nil {
		return r.Queue.QueueId
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
	if r.Queue != nil && r.Queue.MemberId != nil {
		if *r.Queue.MemberId != 0 { //FIXME
			return r.Queue.MemberId
		}
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
	Cause          string         `json:"cause"`
	Payload        *CallVariables `json:"payload"`
	SipCode        *int           `json:"sip"`
	OriginSuccess  *bool          `json:"originate_success"`
	HangupBy       *string        `json:"hangup_by"`
	Tags           []string       `json:"tags"`
	AmdResult      *string        `json:"amd_result"`
	RecordStart    *int64         `json:"record_start,string"`
	RecordStop     *int64         `json:"record_stop,string"`
	TalkSec        *float32       `json:"talk_sec,string"`
	AmdAiResult    *string        `json:"amd_ai_result"`
	AmdAiResultLog []string       `json:"amd_ai_logs"`
	AmdAiPositive  *bool          `json:"amd_ai_positive"`
}

type CallActionSTT struct {
	CallAction
	Transcript string `json:"transcript"`
}

func (h *CallActionHangup) VariablesToJson() []byte {
	if h.Payload == nil {
		return []byte("{}") //FIXME
	}
	data, _ := json.Marshal(h.Payload)
	return data
}

func (h *CallActionHangup) Parameters() []byte {
	if h.RecordStart == nil && h.RecordStop == nil {
		return []byte("{}") //FIXME
	}

	res := make(map[string]interface{})
	if h.RecordStop != nil {
		res["record_stop"] = *h.RecordStop
	}
	if h.RecordStart != nil {
		res["record_start"] = *h.RecordStart
	}

	b, _ := json.Marshal(res)
	return b
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
	IsTransfer() bool
	Direction() CallDirection
	Destination() string
	SetDomainName(name string)
	DomainName() string
	Dump()
	IVRQueueId() *int
	TransferSchemaId() *int

	SetTransferAfterBridge(ctx context.Context, schemaId int) (Response, *AppError)

	SetAll(ctx context.Context, vars Variables) (Response, *AppError)
	SetNoLocal(ctx context.Context, vars Variables) (Response, *AppError)
	UnSet(ctx context.Context, name string) (Response, *AppError)

	RingReady(ctx context.Context) (Response, *AppError)
	PreAnswer(ctx context.Context) (Response, *AppError)
	Answer(ctx context.Context) (Response, *AppError)
	Echo(ctx context.Context, delay int) (Response, *AppError)
	Hangup(ctx context.Context, cause string) (Response, *AppError)
	HangupNoRoute(ctx context.Context) (Response, *AppError)
	HangupAppErr(ctx context.Context) (Response, *AppError)
	Bridge(ctx context.Context, call Call, strategy string, vars map[string]string, endpoints []*Endpoint, codec []string, hook chan struct{}, pickup string) (Response, *AppError)
	Sleep(ctx context.Context, delay int) (Response, *AppError)
	//Voice(ctx context.Context, delay int) (Response, *AppError)
	Conference(ctx context.Context, name, profile, pin string, tags []string) (Response, *AppError)
	RecordFile(ctx context.Context, name, format string, maxSec, silenceThresh, silenceHits int) (Response, *AppError)
	RecordSession(ctx context.Context, name, format string, minSec int, stereo, bridged, followTransfer bool) (Response, *AppError)
	RecordSessionStop(ctx context.Context, name, format string) (Response, *AppError)
	Export(ctx context.Context, vars []string) (Response, *AppError)
	FlushDTMF(ctx context.Context) (Response, *AppError)
	StartDTMF(ctx context.Context) (Response, *AppError)
	StopDTMF(ctx context.Context) (Response, *AppError)
	Park(ctx context.Context, name string, in bool, lotFrom, lotTo string) (Response, *AppError)
	Playback(ctx context.Context, files []*PlaybackFile) (Response, *AppError)
	PlaybackAndGetDigits(ctx context.Context, files []*PlaybackFile, params *PlaybackDigits) (Response, *AppError)
	PlaybackUrl(ctx context.Context, url string) (Response, *AppError)
	PlaybackUrlAndGetDigits(ctx context.Context, fileString string, params *PlaybackDigits) (Response, *AppError)
	TTS(ctx context.Context, path string, digits *PlaybackDigits, timeout int) (Response, *AppError)
	TTSOpus(ctx context.Context, path string, digits *PlaybackDigits, timeout int) (Response, *AppError)

	Redirect(ctx context.Context, uri []string) (Response, *AppError)
	SetSounds(ctx context.Context, lang, voice string) (Response, *AppError)
	ScheduleHangup(ctx context.Context, sec int, cause string) (Response, *AppError)
	Ringback(ctx context.Context, export bool, call, hold, transfer *PlaybackFile) (Response, *AppError)

	DumpExportVariables() map[string]string
	Queue(ctx context.Context, ringFile string) (Response, *AppError)
	Intercept(ctx context.Context, id string) (Response, *AppError)
	GetVariable(string) string

	Amd(ctx context.Context, params AmdParameters) (Response, *AppError)
	AmdML(ctx context.Context, params AmdMLParameters) (Response, *AppError)

	Pickup(ctx context.Context, name string) (Response, *AppError)
	PickupHash(name string) string

	GoogleTranscribe(ctx context.Context) (Response, *AppError)
	GoogleTranscribeStop(ctx context.Context) (Response, *AppError)

	UpdateCid(ctx context.Context, name, number *string) (Response, *AppError)
	Push(ctx context.Context, name, tag string) (Response, *AppError)
	Cv(ctx context.Context) (Response, *AppError)
	Stopped() bool

	SetQueueCancel(cancel context.CancelFunc) bool
	CancelQueue() bool
	HangupCause() string
}

type PlaybackFile struct {
	Type *string `json:"type"`
	Id   *int    `json:"id"`
	Name *string `json:"name"`
}

type TTS struct {
	Key   string `json:"key"`
	Token string `json:"token"`

	Provider   string `json:"provider"`
	Text       string `json:"text"`
	TextType   string `json:"textType"`
	Terminator string `json:"terminator"`
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

type GetSpeech struct {
	Timeout int `json:"timeout"`
}

type PlaybackArgs struct {
	Files      []*PlaybackFile `json:"files"`
	Terminator string          `json:"terminator" def:"#"`
	GetDigits  *PlaybackDigits `json:"getDigits"`
	GetSpeech  *GetSpeech      `json:"getSpeech"`
}
