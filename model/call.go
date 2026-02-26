package model

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/webitel/wlog"

	eng "github.com/webitel/flow_manager/gen/engine"
)

const CallVariableSchemaIds = "wbt_schema_ids"

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

type CallDirection string

const (
	CallExchange       = "call"
	OpensipsExchange   = "opensips"
	ChatExchange       = "chat"
	FlowExchange       = "flow"
	CallEventQueueName = "workflow-call"
	FlowExecQueueName  = "workflow-exec"
	IMQueueNamePrefix  = "im-delivery.workflow-processor.v1"
	IMExchange         = "im_delivery.broadcast"
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

type OutboundCallRequest *eng.CreateCallRequest

type MissedCall struct {
	DomainId int64  `json:"domain_id" db:"domain_id"`
	Id       string `json:"id" db:"id"`
	UserId   int64  `json:"user_id" db:"user_id"`
}

type CallAction struct {
	Id        string `json:"id"`
	AppId     string `json:"app_id"`
	DomainId  int64  `json:"domain_id,string"`
	Timestamp int64  `json:"timestamp,string"`
	Event     string `json:"event"`
}

type CallActionData struct {
	CallAction
	Data   *string `json:"data,omitempty"`
	parsed any     `json:"-"`
}

type CallActionDataWithUser struct {
	CallActionData
	UserId string `json:"user_id" db:"user_id,omitempty"`
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

type RTPAggregate struct {
	Average float32 `json:"average"`
	Min     float32 `json:"min"`
	MinAt   float32 `json:"min_at"`
	Max     float32 `json:"max"`
	MaxAt   float32 `json:"max_at"`
}

type RtpStats struct {
	Mos        RTPAggregate `json:"mos"`
	Jitter     RTPAggregate `json:"jitter"`
	RoundTrip  RTPAggregate `json:"roundtrip"`
	PacketLoss RTPAggregate `json:"packetloss"`
}

type CallMediaStats struct {
	SipId string   `json:"call_id"`
	RTP   RtpStats `json:"rtp"`
}

type QueueInfo struct {
	QueueId   *int   `json:"queue_id,string"`
	AttemptId int64  `json:"attempt_id,string"`
	TeamId    *int   `json:"team_id,string"`
	AgentId   *int   `json:"agent_id,string"`
	MemberId  *int64 `json:"member_id,string"`
}

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
	arr := make([]string, 0, 3)
	if r.SipId != nil {
		arr = append(arr, fmt.Sprintf(`"sip_id":"%s"`, *r.SipId))
	} else {
		arr = append(arr, fmt.Sprintf(`"sip_id":"%s"`, r.Id))
	}

	if r.Heartbeat > 0 {
		arr = append(arr, fmt.Sprintf(`"heartbeat":%d`, r.Heartbeat))
	}
	if r.Video != "" {
		arr = append(arr, fmt.Sprintf(`"video":"%s"`, r.Video))
	}
	if r.MeetingId != "" {
		arr = append(arr, fmt.Sprintf(`"meeting_id":"%s"`, r.MeetingId))
	}

	return []byte(`{` + strings.Join(arr, ",") + `}`)
}

type CallActionActive struct {
	CallAction
}

type CallActionHold struct {
	CallAction
}

type CallActionHeartbeat struct {
	CallAction
}

type CallActionBridge struct {
	CallAction
	BridgedId string `json:"bridged_id"`
	To        CallEndpoint
}

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

type CallActionSTT struct {
	CallAction
	Transcript string `json:"transcript"`
}

type CallActionTranscript struct {
	CallAction
	Transcript any `json:"transcript"`
}

type CallActionMediaStats struct {
	CallAction
	UserId *int64 `json:"user_id,string"`
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
	UserId() int
	// ParentType() *string //TODO transfer logic
	From() *CallEndpoint
	To() *CallEndpoint
	IsTransfer() bool
	IsOriginateRequest() bool
	Direction() CallDirection
	Destination() string
	SetDomainName(name string)
	SetSchemaId(id int) *AppError
	DomainName() string
	Dump()
	IVRQueueId() *int
	TransferSchemaId() *int
	TransferQueueId() int
	IsBlindTransferQueue() bool
	TransferAgentId() int
	SetTransferFromId()
	MeetingId() string

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
	// Voice(ctx context.Context, delay int) (Response, *AppError)
	Conference(ctx context.Context, name, profile, pin string, tags []string) (Response, *AppError)
	RecordFile(ctx context.Context, name, format string, maxSec, silenceThresh, silenceHits int) (Response, *AppError)
	SendFileToAi(ctx context.Context, url string, m map[string]string, format string, maxSec, silenceThresh, silenceHits int) (Response, *AppError)
	RecordSession(ctx context.Context, name, format string, minSec int, stereo, bridged, followTransfer bool) (Response, *AppError)
	RecordSessionStop(ctx context.Context, name, format string) (Response, *AppError)
	Export(ctx context.Context, vars []string) (Response, *AppError)
	FlushDTMF(ctx context.Context) (Response, *AppError)
	StartDTMF(ctx context.Context) (Response, *AppError)
	StopDTMF(ctx context.Context) (Response, *AppError)
	Park(ctx context.Context, name string, in bool, lotFrom, lotTo string) (Response, *AppError)
	Playback(ctx context.Context, files []*PlaybackFile) (Response, *AppError)
	Say(ctx context.Context, val string) (Response, *AppError)
	PlaybackAndGetDigits(ctx context.Context, files []*PlaybackFile, params *PlaybackDigits) (Response, *AppError)
	PlaybackUrl(ctx context.Context, url string) (Response, *AppError)
	PlaybackUrlAndGetDigits(ctx context.Context, fileString string, params *PlaybackDigits) (Response, *AppError)
	PushSpeechMessage(msg SpeechMessage)
	SpeechMessages(limit int) []SpeechMessage

	TTS(ctx context.Context, path string, tts TTSSettings, digits *PlaybackDigits, timeout int) (Response, *AppError)
	TTSOpus(ctx context.Context, path string, digits *PlaybackDigits, timeout int) (Response, *AppError)

	Redirect(ctx context.Context, uri []string) (Response, *AppError)
	SetSounds(ctx context.Context, lang, voice string) (Response, *AppError)
	ScheduleHangup(ctx context.Context, sec int, cause string) (Response, *AppError)
	Ringback(ctx context.Context, export bool, call, hold, transfer *PlaybackFile) (Response, *AppError)

	DumpExportVariables() map[string]string
	ClearExportVariables()

	Queue(ctx context.Context, ringFile string) (Response, *AppError)
	Intercept(ctx context.Context, id string) (Response, *AppError)
	GetVariable(string) string

	Amd(ctx context.Context, params AmdParameters) (Response, *AppError)
	AmdML(ctx context.Context, params AmdMLParameters) (Response, *AppError)

	Pickup(ctx context.Context, name string) (Response, *AppError)
	PickupHash(name string) string

	StartRecognize(ctx context.Context, connection, dialogId string, rate, vadTimeout int) (Response, *AppError)
	StopRecognize(ctx context.Context) (Response, *AppError)

	GoogleTranscribe(ctx context.Context, config *GetSpeech) (Response, *AppError)
	GoogleTranscribeStop(ctx context.Context) (Response, *AppError)
	RefreshVars(ctx context.Context) (Response, *AppError)

	UpdateCid(ctx context.Context, name, number, destination *string) (Response, *AppError)
	Push(ctx context.Context, name, tag string) (Response, *AppError)
	Cv(ctx context.Context) (Response, *AppError)
	Stopped() bool

	SetQueueCancel(cancel context.CancelFunc) bool
	CancelQueue() bool
	InQueue() bool
	HangupCause() string

	GetContactId() int
	BackgroundPlayback(ctx context.Context, file *PlaybackFile, name string, volumeReduction int) (Response, *AppError)
	BackgroundPlaybackStop(ctx context.Context, name string) (Response, *AppError)
	Bot(ctx context.Context, conn string, rate int, startMessage string, vars map[string]string) (Response, *AppError)
	Update(ctx context.Context) (Response, *AppError)
}

type PlaybackFile struct {
	Type *string         `json:"type"`
	Id   *int            `json:"id"`
	Name *string         `json:"name"`
	Args *map[string]any `json:"args"`
	TTS  *TTSSettings    `json:"tts"`
}

type HttpFileArgs struct {
	Url      string            `json:"url,omitempty"`
	FileType string            `json:"fileType,omitempty"`
	Method   string            `json:"method,omitempty"`
	Headers  map[string]string `json:"headers,omitempty"`
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

type SpeechMessage struct {
	Question   string
	Answer     string
	Final      bool
	Confidence int
}

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

type PlaybackArgs struct {
	Files      []*PlaybackFile `json:"files"`
	Terminator string          `json:"terminator" def:"#"`
	GetDigits  *PlaybackDigits `json:"getDigits"`
	GetSpeech  *GetSpeech      `json:"getSpeech"`
}
