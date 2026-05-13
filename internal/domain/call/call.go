package call

// moved from model/call.go — see model/call.go for re-export aliases

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/webitel/wlog"

	"github.com/webitel/flow_manager/internal/domain/flow"
)

const CallVariableSchemaIds = "wbt_schema_ids"

// CallResponse is the result of a call application execution.
type CallResponse struct {
	Status string
}

var (
	CallResponseOK    = &CallResponse{"SUCCESS"}
	CallResponseError = &CallResponse{"ERROR"}
)

func (r CallResponse) String() string {
	return r.Status
}

// CallDirection indicates whether the call is inbound or outbound.
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

const (
	CallActionRingingName    = "ringing"
	CallActionActiveName     = "active"
	CallActionBridgeName     = "bridge"
	CallActionHoldName       = "hold"
	CallActionDtmfName       = "dtmf"
	CallActionSTTName        = "stt"
	CallActionHangupName     = "hangup"
	CallActionHeartbeatName  = "heartbeat"
	CallActionTranscriptName = "transcript"
	CallActionStatsName      = "stats"
)

// OutboundCallRequest describes a request to initiate an outbound call.
type OutboundCallRequest struct {
	From        *OutboundCallEndpoint `json:"from"`
	To          *OutboundCallEndpoint `json:"to"`
	Destination string                `json:"destination"`
	Params      *OutboundCallParams   `json:"params"`
	DomainID    int64                 `json:"domainId"`
}

// OutboundCallEndpoint identifies one leg of an outbound call.
type OutboundCallEndpoint struct {
	AppId     string `json:"appId"`
	Type      string `json:"type"`
	Id        int64  `json:"id"`
	Extension string `json:"extension"`
}

// OutboundCallParams holds optional parameters for an outbound call.
type OutboundCallParams struct {
	Timeout           int32             `json:"timeout"`
	Audio             bool              `json:"audio"`
	Video             bool              `json:"video"`
	Screen            bool              `json:"screen"`
	Record            bool              `json:"record"`
	Variables         map[string]string `json:"variables"`
	Display           string            `json:"display"`
	DisableStun       bool              `json:"disableStun"`
	CancelDistribute  bool              `json:"cancelDistribute"`
	IsOnline          bool              `json:"isOnline"`
	DisableAutoAnswer bool              `json:"disableAutoAnswer"`
	HideNumber        bool              `json:"hideNumber"`
	ContactId         int64             `json:"contactId"`
}

// MissedCall records a call that was not answered.
type MissedCall struct {
	DomainId int64  `json:"domain_id" db:"domain_id"`
	Id       string `json:"id" db:"id"`
	UserId   int64  `json:"user_id" db:"user_id"`
}

// CallAction is the base event payload emitted by the media engine.
type CallAction struct {
	Id        string `json:"id"`
	AppId     string `json:"app_id"`
	DomainId  int64  `json:"domain_id,string"`
	Timestamp int64  `json:"timestamp,string"`
	Event     string `json:"event"`
}

// CallActionData wraps a CallAction with optional raw JSON payload.
type CallActionData struct {
	CallAction
	Data   *string `json:"data,omitempty"`
	parsed any     `json:"-"`
}

// CallActionDataWithUser extends CallActionData with a user identifier.
type CallActionDataWithUser struct {
	CallActionData
	UserId string `json:"user_id" db:"user_id,omitempty"`
}

// CallEndpoint identifies the party on one leg of a call.
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

func (e *CallEndpoint) String() string {
	if e == nil {
		return "empty"
	}
	return fmt.Sprintf("type: %s number: %s name: \"%s\" id: %s", e.Type, e.Number, e.Name, e.Id)
}

func (c CallEndpoint) IntId() *int {
	if r, e := strconv.Atoi(c.Id); e != nil {
		return nil
	} else {
		v := r
		return &v
	}
}

// RTPAggregate holds aggregate RTP statistics.
type RTPAggregate struct {
	Average float32 `json:"average"`
	Min     float32 `json:"min"`
	MinAt   float32 `json:"min_at"`
	Max     float32 `json:"max"`
	MaxAt   float32 `json:"max_at"`
}

// RtpStats holds detailed RTP statistics.
type RtpStats struct {
	Mos        RTPAggregate `json:"mos"`
	Jitter     RTPAggregate `json:"jitter"`
	RoundTrip  RTPAggregate `json:"roundtrip"`
	PacketLoss RTPAggregate `json:"packetloss"`
}

// CallMediaStats holds media quality statistics for one call leg.
type CallMediaStats struct {
	SipId  string   `json:"call_id"`
	UserId *int64   `json:"user_id,string"`
	RTP    RtpStats `json:"rtp"`
}

// QueueInfo holds call-centre queue placement information.
type QueueInfo struct {
	QueueId   *int   `json:"queue_id,string"`
	AttemptId int64  `json:"attempt_id,string"`
	TeamId    *int   `json:"team_id,string"`
	AgentId   *int   `json:"agent_id,string"`
	MemberId  *int64 `json:"member_id,string"`
}

// CallActionInfo contains details about a call leg at the ringing event.
type CallActionInfo struct {
	GatewayId       *int           `json:"gateway_id"`
	UserId          *int           `json:"user_id"`
	Direction       string         `json:"direction"`
	Destination     string         `json:"destination"`
	DestinationName *string        `json:"destination_name"`
	From            *CallEndpoint  `json:"from"`
	To              *CallEndpoint  `json:"to"`
	ParentId        *string        `json:"parent_id"`
	Payload         *CallVariables `json:"payload"`
	Queue           *QueueInfo     `json:"queue"`
	GranteeId       *int           `json:"grantee_id"`
	SipId           *string        `json:"sip_id"`
	Heartbeat       int            `json:"heartbeat,omitempty"`
	Video           string         `json:"video,omitempty"`
	MeetingId       string         `json:"meeting_id,omitempty"`
	ContactId       *int64         `json:"contact_id"`
	HideNumber      *bool          `json:"hide_number,omitempty"`
}

// CallActionRinging is emitted when a call starts ringing.
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
		if *r.Queue.MemberId != 0 { // FIXME
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

func (r *CallActionRinging) GetParams() []byte {
	res := make(map[string]any)

	if r.SipId != nil {
		res["sip_id"] = *r.SipId
	} else {
		res["sip_id"] = r.Id
	}

	if r.Heartbeat > 0 {
		res["heartbeat"] = r.Heartbeat
	}
	if r.Video != "" {
		res["video"] = r.Video
	}
	if r.MeetingId != "" {
		res["meeting_id"] = r.MeetingId
	}
	if r.HideNumber != nil {
		res["hide_number"] = *r.HideNumber
	}

	data, _ := json.Marshal(res)
	return data
}

// CallActionActive is emitted when a call becomes active (answered).
type CallActionActive struct {
	CallAction
}

// CallActionHold is emitted when a call is placed on hold.
type CallActionHold struct {
	CallAction
}

// CallActionHeartbeat is a periodic keep-alive event.
type CallActionHeartbeat struct {
	CallAction
}

// CallActionBridge is emitted when two call legs are bridged.
type CallActionBridge struct {
	CallAction
	BridgedId string `json:"bridged_id"`
	To        CallEndpoint
}

// CallActionHangup is emitted when a call ends.
type CallActionHangup struct {
	CallAction
	Cause          string         `json:"cause"`
	Payload        *CallVariables `json:"payload"`
	SipCode        *int           `json:"sip"`
	SipId          *string        `json:"sip_id"`
	OriginSuccess  *bool          `json:"originate_success"`
	HangupBy       *string        `json:"hangup_by"`
	Tags           []string       `json:"tags"`
	AmdResult      *string        `json:"amd_result"`
	AmdCause       *string        `json:"amd_cause"`
	RecordStart    *int64         `json:"record_start,string"`
	RecordStop     *int64         `json:"record_stop,string"`
	TalkSec        *float32       `json:"talk_sec,string"`
	AmdAiResult    *string        `json:"amd_ai_result"`
	AmdAiResultLog []string       `json:"amd_ai_logs"`
	AmdAiPositive  *bool          `json:"amd_ai_positive"`
	CDR            *bool          `json:"cdr"`
	SchemaIds      []int          `json:"schema_ids"`
	HangupPhrase   *string        `json:"hangup_phrase,omitempty"`
	TransferFrom   *string        `json:"transfer_from,omitempty"`
}

// CallActionSTT is emitted with a speech-to-text result.
type CallActionSTT struct {
	CallAction
	Transcript string `json:"transcript"`
}

// CallActionTranscript is emitted with a transcript event.
type CallActionTranscript struct {
	CallAction
	Transcript any `json:"transcript"`
}

// CallActionMediaStats is emitted with media quality statistics.
type CallActionMediaStats struct {
	CallAction
	CallMediaStats
}

func (h *CallActionHangup) VariablesToJson() []byte {
	if h.Payload == nil {
		return []byte("{}") // FIXME
	}
	data, _ := json.Marshal(h.Payload)
	return data
}

func (h *CallActionHangup) Parameters() []byte {
	if h.RecordStart == nil && h.RecordStop == nil && h.AmdCause == nil {
		return []byte("{}") // FIXME
	}

	res := make(map[string]any)
	if h.RecordStop != nil {
		res["record_stop"] = *h.RecordStop
	}
	if h.RecordStart != nil {
		res["record_start"] = *h.RecordStart
	}

	if h.AmdCause != nil {
		res["amd_cause"] = *h.AmdCause
	}

	b, _ := json.Marshal(res)
	return b
}

func (h *CallActionHangup) AmdJson() []byte {
	res := make(map[string]any)
	if h.AmdResult != nil {
		res["result"] = *h.AmdResult
	}
	if h.AmdCause != nil {
		res["cause"] = *h.AmdCause
	}

	// or AI
	if h.AmdAiResult != nil {
		res["result"] = *h.AmdAiResult
	}
	if h.AmdAiResultLog != nil {
		res["log"] = h.AmdAiResultLog
	}
	if h.AmdAiPositive != nil {
		res["positive"] = *h.AmdAiPositive
	}

	if len(res) == 0 {
		return nil
	}

	b, _ := json.Marshal(res)
	return b
}

// CallVariables is a generic map of call-scoped variables.
type CallVariables map[string]any

func (v *CallVariables) ToMapJson() []byte {
	if v != nil {
		d, e := json.Marshal(v)
		if e == nil {
			return d
		}
	}
	return []byte("{}")
}

// GetEvent parses and returns the strongly-typed event struct for this action.
func (c *CallActionData) GetEvent() any {
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
	case CallActionHeartbeatName:
		c.parsed = &CallActionHeartbeat{
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
	case CallActionTranscriptName:
		c.parsed = &CallActionTranscript{
			CallAction: c.CallAction,
		}
	case CallActionStatsName:
		c.parsed = &CallActionMediaStats{
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

// Call is the runtime interface for an active call connection.
type Call interface {
	flow.Connection
	UserId() int
	// ParentType() *string //TODO transfer logic
	From() *CallEndpoint
	To() *CallEndpoint
	IsTransfer() bool
	IsOriginateRequest() bool
	Direction() CallDirection
	Destination() string
	SetDomainName(name string)
	SetSchemaId(id int) error
	DomainName() string
	Dump()
	IVRQueueId() *int
	TransferSchemaId() *int
	TransferQueueId() int
	IsBlindTransferQueue() bool
	TransferAgentId() int
	SetTransferFromId()
	MeetingId() string

	SetTransferAfterBridge(ctx context.Context, schemaId int) (flow.Response, error)

	SetAll(ctx context.Context, vars flow.Variables) (flow.Response, error)
	SetNoLocal(ctx context.Context, vars flow.Variables) (flow.Response, error)
	UnSet(ctx context.Context, name string) (flow.Response, error)

	RingReady(ctx context.Context) (flow.Response, error)
	PreAnswer(ctx context.Context) (flow.Response, error)
	Answer(ctx context.Context) (flow.Response, error)
	Echo(ctx context.Context, delay int) (flow.Response, error)
	Hangup(ctx context.Context, cause string) (flow.Response, error)
	HangupNoRoute(ctx context.Context) (flow.Response, error)
	HangupAppErr(ctx context.Context) (flow.Response, error)
	Bridge(ctx context.Context, call Call, strategy string, vars map[string]string, endpoints []*Endpoint, codec []string, hook chan struct{}, pickup string) (flow.Response, error)
	Sleep(ctx context.Context, delay int) (flow.Response, error)
	// Voice(ctx context.Context, delay int) (flow.Response, error)
	Conference(ctx context.Context, name, profile, pin string, tags []string) (flow.Response, error)
	RecordFile(ctx context.Context, name, format string, maxSec, silenceThresh, silenceHits int) (flow.Response, error)
	SendFileToAi(ctx context.Context, url string, m map[string]string, format string, maxSec, silenceThresh, silenceHits int) (flow.Response, error)
	RecordSession(ctx context.Context, name, format string, minSec int, stereo, bridged, followTransfer bool) (flow.Response, error)
	RecordSessionStop(ctx context.Context, name, format string) (flow.Response, error)
	Export(ctx context.Context, vars []string) (flow.Response, error)
	FlushDTMF(ctx context.Context) (flow.Response, error)
	StartDTMF(ctx context.Context) (flow.Response, error)
	StopDTMF(ctx context.Context) (flow.Response, error)
	Park(ctx context.Context, name string, in bool, lotFrom, lotTo string) (flow.Response, error)
	Playback(ctx context.Context, files []*PlaybackFile) (flow.Response, error)
	Say(ctx context.Context, val string) (flow.Response, error)
	PlaybackAndGetDigits(ctx context.Context, files []*PlaybackFile, params *PlaybackDigits) (flow.Response, error)
	PlaybackUrl(ctx context.Context, url string) (flow.Response, error)
	PlaybackUrlAndGetDigits(ctx context.Context, fileString string, params *PlaybackDigits) (flow.Response, error)
	PushSpeechMessage(msg SpeechMessage)
	SpeechMessages(limit int) []SpeechMessage

	TTS(ctx context.Context, path string, tts TTSSettings, digits *PlaybackDigits, timeout int) (flow.Response, error)
	TTSOpus(ctx context.Context, path string, digits *PlaybackDigits, timeout int) (flow.Response, error)

	Redirect(ctx context.Context, uri []string) (flow.Response, error)
	SetSounds(ctx context.Context, lang, voice string) (flow.Response, error)
	ScheduleHangup(ctx context.Context, sec int, cause string) (flow.Response, error)
	Ringback(ctx context.Context, export bool, call, hold, transfer *PlaybackFile) (flow.Response, error)

	DumpExportVariables() map[string]string
	ClearExportVariables()

	Queue(ctx context.Context, ringFile string) (flow.Response, error)
	Intercept(ctx context.Context, id string) (flow.Response, error)
	GetVariable(string) string

	Amd(ctx context.Context, params AmdParameters) (flow.Response, error)
	AmdML(ctx context.Context, params AmdMLParameters) (flow.Response, error)

	Pickup(ctx context.Context, name string) (flow.Response, error)
	PickupHash(name string) string

	StartRecognize(ctx context.Context, connection, dialogId string, rate, vadTimeout int) (flow.Response, error)
	StopRecognize(ctx context.Context) (flow.Response, error)

	GoogleTranscribe(ctx context.Context, config *GetSpeech) (flow.Response, error)
	GoogleTranscribeStop(ctx context.Context) (flow.Response, error)
	RefreshVars(ctx context.Context) (flow.Response, error)

	UpdateCid(ctx context.Context, name, number, destination *string) (flow.Response, error)
	Push(ctx context.Context, name, tag string) (flow.Response, error)
	Cv(ctx context.Context) (flow.Response, error)
	Stopped() bool

	SetQueueCancel(cancel context.CancelFunc) bool
	CancelQueue() bool
	InQueue() bool
	HangupCause() string

	GetContactId() int
	BackgroundPlayback(ctx context.Context, file *PlaybackFile, name string, volumeReduction int) (flow.Response, error)
	BackgroundPlaybackStop(ctx context.Context, name string) (flow.Response, error)
	Bot(ctx context.Context, conn string, rate int, startMessage string, vars map[string]string) (flow.Response, error)
	Update(ctx context.Context) (flow.Response, error)
}

// PlaybackFile describes a single audio file for playback.
type PlaybackFile struct {
	Type *string         `json:"type"`
	Id   *int            `json:"id"`
	Name *string         `json:"name"`
	Args *map[string]any `json:"args"`
	TTS  *TTSSettings    `json:"tts"`
}

// HttpFileArgs holds parameters for an HTTP-based file resource.
type HttpFileArgs struct {
	Url      string            `json:"url,omitempty"`
	FileType string            `json:"fileType,omitempty"`
	Method   string            `json:"method,omitempty"`
	Headers  map[string]string `json:"headers,omitempty"`
}

// TTS holds simple key/token credentials for a TTS request.
type TTS struct {
	Key   string `json:"key"`
	Token string `json:"token"`

	Provider   string `json:"provider"`
	Text       string `json:"text"`
	TextType   string `json:"textType"`
	Terminator string `json:"terminator"`
}

// PlaybackDigits holds parameters for digit collection during playback.
type PlaybackDigits struct {
	SetVar       *string `json:"setVar"`
	Min          *int    `json:"min" def:"1"`
	Max          *int    `json:"max"`
	Tries        *int    `json:"tries"`
	Timeout      *int    `json:"timeout"`
	FlushDtmf    bool    `json:"flushDTMF"`
	Regexp       *string `json:"regexp"`
	DigitTimeout *int    `json:"digitTimeout"`
	Terminators  string  `json:"terminators"`
}

// SpeechMessage holds a single speech recognition turn.
type SpeechMessage struct {
	Question   string
	Answer     string
	Final      bool
	Confidence int
}

// GetSpeech holds parameters for speech recognition.
type GetSpeech struct {
	Background *struct {
		Name            string
		File            *PlaybackFile `json:"file" json:"file,omitempty"`
		VolumeReduction int           `json:"volumeReduction" json:"volume_reduction,omitempty"`
	}
	SetVar  string `json:"setVar"`
	Timeout int    `json:"timeout"` // якщо не було відповіді від гугла isFinal
	/*
		vadTimeout, це час в мілісекундах, якщо протягом цього буде тишина то ми виходимо з
		is_final=true, transcript=""
					RECOGNIZER_VAD_SILENCE_MS - 400
				RECOGNIZER_VAD_VOICE_MS - 150
				RECOGNIZER_VAD_THRESH - 200
			1. RECOGNIZER_VAD_SILENCE_MS - це параметр, який визначає тривалість тиші (у мілісекундах),
				після якої система вважає, що голосова активність завершилася і почалася тиша. Простіше кажучи, це "таймер тиші",
			який допомагає уникнути помилкових переходів між голосом і тишею через короткі паузи в розмові.
			getSpeech.vadTimeout - це якщо протягом цього буде визначено як тиша, тоді виходимо з блоку СТТ

			Як працює на прикладі:

			Якщо значення RECOGNIZER_VAD_SILENCE_MS = 400:
			Система чекатиме 400 мс тиші після завершення голосу, щоб зрозуміти, що голос дійсно закінчився.
			Якщо пауза в розмові буде коротшою (наприклад, 300 мс), система продовжить вважати, що це частина голосової активності.


			2. RECOGNIZER_VAD_THRESH — це параметр, який визначає поріг чутливості системи до енергії звукового сигналу.
			Система використовує це значення, щоб вирішити, чи є звук голосом або фоновим шумом.
			Параметр RECOGNIZER_VAD_THRESH задається у цифрових одиницях рівня амплітуди сигналу.
	*/
	VadTimeout int `json:"vadTimeout"`
	/*
		виключити переривання плейбеку якщо отримали isFinal
	*/
	DisableBreakFinal bool `json:"disableBreakFinal"`
	/*
		breakFinalOnTimeout, якщо true а також disableBreakFinal = true,
		тоді ми очікуємо завершення програвання файлу і у час timeout очікуємо розпізнавання голосу,
		якщо він буде, тоді тишина перерветься.
	*/
	BreakFinalOnTimeout bool     `json:"breakFinalOnTimeout"`
	MinWords            int      `json:"minWords"`
	MaxWords            int      `json:"maxWords"`
	BreakStability      float32  `json:"breakStability"`
	Version             string   `json:"version"`             // (v1, v2) V1 default
	Model               string   `json:"model"`               // v2
	Uri                 string   `json:"uri"`                 // v2
	Recognizer          string   `json:"recognizer"`          // v2
	Lang                string   `json:"lang"`                // V2
	Interim             bool     `json:"interim"`             // V2
	SingleUtterance     bool     `json:"singleUtterance"`     // V2
	SeparateRecognition bool     `json:"separateRecognition"` // V2
	MaxAlternatives     int      `json:"maxAlternatives"`     // V2
	ProfanityFilter     bool     `json:"profanityFilter"`     // V2
	WordTime            bool     `json:"wordTime"`            // V2
	Punctuation         bool     `json:"punctuation"`         // V2
	Enhanced            bool     `json:"enhanced"`            // V2
	Hints               string   `json:"hints"`               // V2
	AlternativeLang     []string `json:"alternativeLang"`     // v2
	SampleRate          int      `json:"sampleRate"`          // v2
	Question            string   `json:"question"`

	// v3
	Profile struct {
		Id int32 `json:"id"`
	} `json:"profile"`
	ExtraParams map[string]string `json:"extraParams"`
}

// PlaybackArgs groups all playback parameters.
type PlaybackArgs struct {
	Files      []*PlaybackFile `json:"files"`
	Terminator string          `json:"terminator" def:"#"`
	GetDigits  *PlaybackDigits `json:"getDigits"`
	GetSpeech  *GetSpeech      `json:"getSpeech"`
}
