package fs

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"

	"github.com/tidwall/gjson"

	uuid "github.com/satori/go.uuid"
	"github.com/webitel/flow_manager/model"
	"github.com/webitel/flow_manager/providers/fs/eventsocket"
	"github.com/webitel/wlog"
)

const (
	HEADER_DOMAIN_ID  = "variable_sip_h_X-Webitel-Domain-Id"
	HEADER_USER_ID    = "variable_sip_h_X-Webitel-User-Id"
	HEADER_GATEWAY_ID = "variable_sip_h_X-Webitel-Gateway-Id"

	HEADER_CONTEXT_NAME              = "Channel-Context"
	HEADER_ID_NAME                   = "Unique-ID"
	HEADER_DIRECTION_NAME            = "variable_sip_h_X-Webitel-Direction"
	HEADER_EVENT_NAME                = "Event-Name"
	HEADER_EVENT_ID_NAME             = "Event-UUID"
	HEADER_CORE_ID_NAME              = "Core-UUID"
	HEADER_CORE_NAME                 = "FreeSWITCH-Switchname"
	HEADER_APPLICATION_ID_NAME       = "Application-UUID"
	HEADER_APPLICATION_NAME          = "Application"
	HEADER_APPLICATION_DATA_NAME     = "Application-Data"
	HEADER_APPLICATION_RESPONSE_NAME = "Application-Response"
	HEADER_HANGUP_CAUSE_NAME         = "variable_hangup_cause"
	HEADER_CONTENT_TYPE_NAME         = "Content-Type"
	HEADER_CONTENT_DISPOSITION_NAME  = "Content-Disposition"

	HEADER_CHANNEL_DESTINATION_NAME  = "Channel-Destination-Number"
	HEADER_CALLER_DESTINATION_NAME   = "Caller-Destination-Number"
	HEADER_VARIABLE_DESTINATION_NAME = "variable_destination_number"
)

// коли юзр то не прцюэ трансфер на оцынку
const (
	UsrVarPrefix = ""
)

var errExecuteAfterHangup = model.NewAppError("FreeSWITCH", "provider.fs.execute.after_hangup", nil, "not allow after hangup", http.StatusBadRequest)

type Connection struct {
	id               string
	nodeId           string
	nodeName         string
	transfer         bool
	transferFromId   string
	originateRequest bool
	webCall          string
	dialPlan         string
	//context         string
	destination          string
	stopped              bool
	direction            model.CallDirection
	gatewayId            int
	domainId             int64
	domainName           string
	from                 *model.CallEndpoint
	to                   *model.CallEndpoint
	systemDirection      string
	schemaId             *int
	resample             int
	transferSchema       int
	transferQueue        int
	transferAgent        int
	isBlindTransferQueue bool

	userId           int
	disconnected     chan struct{}
	lastEvent        *eventsocket.Event
	connection       *eventsocket.Connection
	callbackMessages map[string]chan *eventsocket.Event
	variables        *model.ThreadSafeStringMap
	hangupCause      string
	exportVariables  []string
	ctx              context.Context
	cancelFn         context.CancelFunc
	hookBridged      chan struct{} //todo
	cancelQueue      context.CancelFunc
	speechMessages   []model.SpeechMessage
	playBackground   int
	sync.RWMutex
	useTTSPrepare bool

	log *wlog.Logger
}

func (c *Connection) Type() model.ConnectionType {
	return model.ConnectionTypeCall
}

func getDirection(str string) model.CallDirection {
	switch str {
	case model.CallDirectionOutbound, "internal":
		return model.CallDirectionOutbound
	default:
		return model.CallDirectionInbound
	}
}

func newConnection(baseConnection *eventsocket.Connection, dump *eventsocket.Event) *Connection {
	ctx, cancel := context.WithCancel(context.TODO())
	connection := &Connection{
		id:               dump.Get(HEADER_ID_NAME),
		nodeId:           dump.Get(HEADER_CORE_ID_NAME),
		nodeName:         dump.Get(HEADER_CORE_NAME),
		ctx:              ctx,
		cancelFn:         cancel,
		dialPlan:         dump.Get(HEADER_CONTEXT_NAME),
		direction:        getDirection(dump.Get(HEADER_DIRECTION_NAME)),
		gatewayId:        getIntFromStr(dump.Get(HEADER_GATEWAY_ID)),
		domainId:         int64(getIntFromStr(dump.Get(HEADER_DOMAIN_ID))),
		userId:           getIntFromStr(dump.Get(HEADER_USER_ID)),
		connection:       baseConnection,
		lastEvent:        dump,
		callbackMessages: make(map[string]chan *eventsocket.Event),
		//disconnected:     make(chan struct{}),
		variables: model.NewThreadSafeStringMap(),
	}
	connection.log = wlog.GlobalLogger().With(
		wlog.Namespace("context"),
		wlog.String("scope", "call"),
		wlog.String("call_id", connection.id),
		wlog.Int("user_id", connection.userId),
		wlog.Int("gateway_id", connection.gatewayId),
		wlog.Int64("domain_id", connection.domainId),
	)
	connection.initIvrQueue(dump)
	connection.initTransferSchema(dump)
	connection.initTransferQueue(dump)
	connection.setCallInfo(dump)
	connection.updateVariablesFromEvent(dump)
	return connection
}

func (c *Connection) Log() *wlog.Logger {
	return c.log
}

func (c *Connection) initIvrQueue(event *eventsocket.Event) {
	s := event.Get("variable_cc_queue_id")
	if s != "" && event.Get("variable_cc_queue_type") == "ivr" {
		if i, err := strconv.Atoi(s); err == nil {
			c.schemaId = &i
		}
	}
}

func (c *Connection) initTransferSchema(event *eventsocket.Event) {
	c.transferSchema, _ = strconv.Atoi(event.Get("variable_transfer_to_schema_id"))
	if c.transferSchema != 0 {
		//c.executeWithContext(c.ctx, "unset", "transfer_to_schema_id")
	}
}

func (c *Connection) initTransferQueue(event *eventsocket.Event) {
	c.transferQueue, _ = strconv.Atoi(event.Get("variable_wbt_bt_queue_id"))
	if c.transferQueue != 0 {
		//c.executeWithContext(c.ctx, "unset", "wbt_bt_queue_id")
	}

	c.transferAgent, _ = strconv.Atoi(event.Get("variable_wbt_bt_agent_id"))
	if c.transferAgent != 0 {
		//c.executeWithContext(c.ctx, "unset", "wbt_bt_queue_id")
	}

	c.isBlindTransferQueue = event.Get("variable_wbt_bt_queue") == "true"
}

func (c *Connection) IsBlindTransferQueue() bool {
	return c.isBlindTransferQueue
}

func (c *Connection) TransferSchemaId() *int {
	if (c.dialPlan == "default" || c.webCall != "") && c.transferSchema != 0 {
		return &c.transferSchema
	}

	return nil
}

func (c *Connection) TransferQueueId() int {
	return c.transferQueue
}

func (c *Connection) TransferAgentId() int {
	return c.transferAgent
}

func (c *Connection) IVRQueueId() *int {
	return c.schemaId
}

func (c *Connection) SetTransferFromId() {
	if c.lastEvent != nil {
		c.transferFromId = c.lastEvent.Get("variable_last_bridge_to")
	}
}

// TODO
func (c *Connection) IsTransfer() bool {
	return c.transfer || c.webCall != ""
}

func (c *Connection) DialPlan() string {
	return c.dialPlan
}

func (c *Connection) Dump() {
	c.lastEvent.PrettyPrint()
}

func (c *Connection) IsOriginateRequest() bool {
	return c.originateRequest
}

func (c *Connection) GetContactId() int {
	if conId, ok := c.Get("wbt_contact_id"); ok {
		contactId, _ := strconv.Atoi(conId)
		return contactId
	}

	return 0
}

func (c *Connection) setCallInfo(dump *eventsocket.Event) {
	direction := dump.Get("variable_sip_h_X-Webitel-Direction")
	isOriginate := dump.Get("variable_sip_h_X-Webitel-Display-Direction") != ""
	c.transfer = dump.Get("variable_transfer_source") != ""
	c.webCall = dump.Get("variable_wbt_web_call")

	if dump.Get("variable_sip_h_X-Webitel-Origin") == "request" && dump.Get("variable_grpc_originate_success") == "true" {
		c.originateRequest = true
	}

	if direction == "internal" {
		if dump.Get("Call-Direction") == "outbound" && !isOriginate {
			direction = "inbound"
		} else {
			direction = "outbound"
		}
	}

	if isOriginate {
		c.destination = dump.Get("variable_effective_callee_id_number")
	} else {
		c.initDestination(dump)
	}
	c.initDestination(dump)
	//dump.PrettyPrint()

	c.from = &model.CallEndpoint{}

	if c.gatewayId != 0 && c.userId == 0 {
		c.from.Type = model.CallEndpointTypeDestination
		c.from.Number = dump.Get("Caller-Caller-ID-Number")
		c.from.Name = dump.Get("Caller-Caller-ID-Name")

		c.to = &model.CallEndpoint{
			Type:   model.CallEndpointTypeGateway,
			Id:     dump.Get("variable_sip_h_X-Webitel-Gateway-Id"),
			Name:   dump.Get("variable_sip_h_X-Webitel-Gateway"),
			Number: c.from.Number,
		}
	} else if c.userId != 0 {
		if direction == "inbound" {
			c.from.Type = model.CallEndpointTypeUser
			c.from.Id = fmt.Sprintf("%d", c.userId)
			c.from.Name = dump.Get("Caller-Caller-ID-Name")
			c.from.Number = dump.Get("Caller-Caller-ID-Number")
		} else {
			c.from.Type = model.CallEndpointTypeUser
			c.from.Id = fmt.Sprintf("%d", c.userId)
			c.from.Name = dump.Get("Caller-Caller-ID-Name")
			c.from.Number = dump.Get("Caller-Caller-ID-Number")
		}
		//fmt.Println(direction)
	} else if c.webCall != "" && c.transferSchema != 0 {
		c.to = &model.CallEndpoint{
			Type:   model.CallEndpointTypeDestination,
			Id:     "",
			Name:   dump.Get("Caller-Orig-Caller-ID-Name"),
			Number: dump.Get("Caller-Orig-Caller-ID-Number"),
		}
	} else {
		c.from.Type = "unknown"
	}
}

func (c *Connection) From() *model.CallEndpoint {
	return c.from
}

func (c *Connection) To() *model.CallEndpoint {
	return c.to
}

func getIntFromStr(str string) int {
	i, _ := strconv.Atoi(str)
	return i
}

func (c *Connection) Id() string {
	return c.id
}

func (c *Connection) DomainId() int64 {
	return c.domainId
}

func (c *Connection) SetDomainName(name string) {
	c.domainName = name
}

func (c *Connection) SetSchemaId(id int) *model.AppError {
	_, err := c.Push(context.Background(), model.CallVariableSchemaIds, strconv.Itoa(id))
	return err
}

func (c *Connection) DomainName() string {
	return c.domainName
}

func (c *Connection) UserId() int {
	return c.userId
}

func (c *Connection) ParseText(text string, ops ...model.ParseOption) string {
	return text
}

func (c *Connection) Context() context.Context {
	return c.ctx
}

func (c *Connection) InboundGatewayId() int {
	return c.gatewayId
}

func (c *Connection) Direction() model.CallDirection {
	return c.direction
}

func (c *Connection) PrintLastEvent() {
	if c.lastEvent != nil {
		c.lastEvent.PrettyPrint()
	}
}

func (c *Connection) Close() *model.AppError {
	c.connection.Close()
	//FIXME
	return nil
}

func (c *Connection) Get(key string) (string, bool) {
	idx := strings.Index(key, ".")

	if idx > 0 {
		root := key[0:idx]
		v, ok := c.get(root)
		if ok {
			return gjson.GetBytes([]byte(v), key[idx+1:]).String(), true
		}
	}

	return c.get(key)
}

func (c *Connection) get(key string) (value string, ok bool) {

	if c.Stopped() {
		if value, ok = c.variables.Load(key); ok {
			return
		}
	}

	if c.lastEvent != nil {
		value = c.lastEvent.Get("variable_" + c.UserVariablePrefix(key))
		if value == "" {
			value = c.lastEvent.Get("variable_" + (key))
		}
		if value == "" {
			var mapKey = ""
			if mapKey, ok = mapVariables[key]; ok {
				value = c.lastEvent.Get(mapKey)
			}
		}

		if value == "" {
			value = c.lastEvent.Get(key)
		}

		if value != "" {
			ok = true
		}
	}
	return
}

func (c *Connection) setDisconnectedVariables(vars model.Variables) (model.Response, *model.AppError) {
	m := make(map[string]string)
	for k, v := range vars {
		m[k] = fmt.Sprintf("%v", v)
	}
	c.variables.UnionMap(m)
	return model.CallResponseOK, nil
}

func escapeFsMultiset(src string) string {
	res := ""
	s := []rune(src)
	l := len(s)

	for i := 0; i < l; i++ {
		if s[i] == '~' && i > 0 && s[i-1] != '\\' {
			res += `\` + string(s[i])
		} else if s[i] == '\'' && i > 0 && s[i-1] != '\\' {
			res += `\\` + string(s[i])
		} else {
			res += string(s[i])
		}
	}

	return res
}

func (c *Connection) setChannelVariables(ctx context.Context, pref string, vars model.Variables) (model.Response, *model.AppError) {
	str := "^^"
	for k, v := range vars {
		str += fmt.Sprintf(`~'%s%s'=%v`, pref, k, escapeFsMultiset(fmt.Sprintf("%v", v)))
	}

	return c.executeWithContext(ctx, "multiset", str)
}

func (c *Connection) setInternal(ctx context.Context, vars model.Variables) (model.Response, *model.AppError) {
	if c.Stopped() {
		return nil, model.NewAppError("Call.setInternal", "call.app.set_internal.stopped", nil, "bad request", http.StatusBadRequest)
	}

	return c.setChannelVariables(ctx, "", vars)
}

func (c *Connection) UserVariablePrefix(name string) string {
	return UsrVarPrefix + name
}

func (c *Connection) Set(ctx context.Context, vars model.Variables) (model.Response, *model.AppError) {
	if len(vars) == 0 {
		return nil, model.NewAppError("Call.Set", "call.app.set.valid.args", nil, "bad request", http.StatusBadRequest)
	}

	if c.Stopped() {
		return c.setDisconnectedVariables(vars)
	} else {
		return c.setChannelVariables(ctx, UsrVarPrefix, vars)
	}
}

func (c *Connection) SetAll(ctx context.Context, vars model.Variables) (model.Response, *model.AppError) {
	var err *model.AppError
	for k, v := range vars {
		if _, err = c.executeWithContext(ctx, "export", fmt.Sprintf(`'%s'='%v'`, c.UserVariablePrefix(k), v)); err != nil {
			return nil, err
		}
	}

	return model.CallResponseOK, nil
}

func (c *Connection) ClearExportVariables() {
	c.Lock()
	c.exportVariables = nil
	c.Unlock()
}

func (c *Connection) DumpExportVariables() map[string]string {
	c.RLock()
	defer c.RUnlock()

	var res = make(map[string]string)
	if len(c.exportVariables) > 0 {
		for _, v := range c.exportVariables {
			res[v], _ = c.Get(v)
		}
	}
	return res
}

func (c *Connection) SetNoLocal(ctx context.Context, vars model.Variables) (model.Response, *model.AppError) {
	var err *model.AppError
	for k, v := range vars {
		if _, err = c.executeWithContext(ctx, "export", fmt.Sprintf(`nolocal:'%s'='%v'`, k, v)); err != nil {
			return nil, err
		}
	}

	return model.CallResponseOK, nil
}

func (c *Connection) initDestination(dump *eventsocket.Event) {
	c.destination = dump.Get(HEADER_CHANNEL_DESTINATION_NAME)
	if c.destination != "" {
		return
	}

	c.destination = dump.Get(HEADER_CALLER_DESTINATION_NAME)
	if c.destination != "" {
		return
	}

	c.destination = dump.Get(HEADER_VARIABLE_DESTINATION_NAME)
	if c.destination != "" {
		return
	}

}

func (c *Connection) Destination() string {
	return c.destination
}

func (c *Connection) NodeId() string {
	return c.nodeId
}

func (c *Connection) Node() string {
	return c.nodeName
}

func (c *Connection) setEvent(event *eventsocket.Event) {
	c.Lock()
	defer c.Unlock()
	if event.Get(HEADER_EVENT_NAME) != "" {
		c.lastEvent = event
		c.updateVariablesFromEvent(event)

		switch event.Get(HEADER_EVENT_NAME) {
		case EVENT_EXECUTE_COMPLETE:
			if s, ok := c.callbackMessages[event.Get(HEADER_APPLICATION_ID_NAME)]; ok {
				delete(c.callbackMessages, event.Get(HEADER_APPLICATION_ID_NAME))
				s <- event
				close(s)
			} else if s, ok := c.callbackMessages[event.Get(HEADER_EVENT_ID_NAME)]; ok {
				delete(c.callbackMessages, event.Get(HEADER_EVENT_ID_NAME))
				s <- event
				close(s)
			}
			c.log.Debug("executed "+event.Get(HEADER_APPLICATION_NAME),
				wlog.String("method", event.Get(HEADER_APPLICATION_NAME)),
				wlog.String("data", event.Get(HEADER_APPLICATION_DATA_NAME)),
				wlog.String("response", event.Get(HEADER_APPLICATION_RESPONSE_NAME)),
			)
		case EVENT_HANGUP_COMPLETE:
			c.hangupCause = event.Get(HEADER_HANGUP_CAUSE_NAME)
			c.log.Debug("hangup", wlog.String("cause", c.hangupCause))
			c.cancelFn()
			//TODO SET DISCONNECT ROUTE
			c.connection.Send("exit")
			c.stopped = true
		case EVENT_BRIDGE:
			c.log.Debug("receive bridge", wlog.String("event_name", EVENT_BRIDGE))
			c.sendHookBridged()
		default:
			c.log.Debug("receive event", wlog.String("event_name", event.Get(HEADER_EVENT_NAME)))
		}
	} else if event.Get(HEADER_CONTENT_TYPE_NAME) == "text/disconnect-notice" && event.Get(HEADER_CONTENT_DISPOSITION_NAME) == "Disconnected" {

	}
}

func (c *Connection) sendHookBridged() {
	if c.hookBridged != nil {
		c.hookBridged <- struct{}{}
		c.closeHookBridge()
		wlog.Debug(fmt.Sprintf("call %s send hook %s", c.Id(), EVENT_BRIDGE))
	}
}

func (c *Connection) setHookBridged(ch chan struct{}) {
	if c.hookBridged == nil {
		c.hookBridged = ch
	}
}

func (c *Connection) closeHookBridge() {
	if c.hookBridged != nil {
		close(c.hookBridged)
		c.hookBridged = nil
	}
}

func (c *Connection) Stopped() bool {
	c.RLock()
	defer c.RUnlock()
	return c.stopped
}

func (c *Connection) Api(cmd string) ([]byte, error) {
	res, err := c.connection.Send(fmt.Sprintf("api %s", cmd))
	if err != nil {
		return []byte(""), err
	}

	return []byte(res.Body), nil
}

func (c *Connection) HangupCause() string {
	c.RLock()
	defer c.RUnlock()
	if c.hangupCause == "" && c.variables != nil {
		if v, ok := c.variables.Load("Hangup-Cause"); ok && v != "" {
			return v
		}
	}
	return c.hangupCause
}

func (c *Connection) executeWithContext(ctx context.Context, app string, args interface{}) (model.Response, *model.AppError) {
	if c.Stopped() {
		return nil, errExecuteAfterHangup
	}

	//c.log.Debug("try execute",
	//	wlog.String("method", app),
	//	wlog.Any("args", args),
	//)

	guid, err := uuid.NewV4()
	if err != nil {
		return nil, model.NewAppError("FreeSWITCH", "provider.fs.execute.gen_uuid", nil, err.Error(), http.StatusInternalServerError)
	}

	e := make(chan *eventsocket.Event, 1)

	c.Lock()
	c.callbackMessages[guid.String()] = e
	c.Unlock()

	_, err = c.connection.SendMsg(eventsocket.MSG{
		"call-command":     "execute",
		"execute-app-name": app,
		"execute-app-arg":  fmt.Sprintf("%v", args),
		"event-lock":       "false",
		"Event-UUID":       guid.String(),
	}, "", "")

	if err != nil {
		return nil, model.NewAppError("FreeSWITCH", "provider.fs.execute.app_error", nil, err.Error(), http.StatusInternalServerError)
	}

	select {
	case <-e:
		return model.CallResponseOK, nil
	case <-ctx.Done():
		return nil, model.NewAppError("FreeSWITCH", "provider.fs.execute.app_error", nil, "cancel", http.StatusInternalServerError)
	}
}

func (c *Connection) executeLoop(app, args string) *model.AppError {
	_, err := c.connection.SendMsg(eventsocket.MSG{
		"call-command":     "execute",
		"execute-app-name": app,
		"execute-app-arg":  fmt.Sprintf("%v", args),
		"event-lock":       "true",
		"loop":             "-1",
	}, "", "")

	if err != nil {
		return model.NewAppError("FreeSWITCH", "provider.fs.execute_loop.app_error", nil, err.Error(), http.StatusInternalServerError)
	}

	return nil

}

func (c *Connection) updateVariablesFromEvent(event *eventsocket.Event) {
	m := make(map[string]string)
	for k, _ := range event.Header {
		m[k] = event.Get(k)
	}
	c.variables.UnionMap(m)
}

func (c *Connection) GetVariable(name string) (value string) {
	c.RLock()
	defer c.RUnlock()
	if c.lastEvent != nil {
		value = c.lastEvent.Get(name)
	}

	return
}

func (c *Connection) WaitForDisconnect1() {
	<-c.disconnected
}

func (c *Connection) SendEvent(m map[string]string, name string) error {
	return c.connection.SendEvent(m, name)
}

func (c *Connection) DumpVariables() map[string]string {
	return c.variables.Data()
}

func (c *Connection) IsSetResample() bool {
	return c.GetVariable("variable_record_sample_rate") != ""
}

func (c *Connection) SetQueueCancel(cancel context.CancelFunc) bool {
	c.Lock()
	defer c.Unlock()

	c.cancelQueue = cancel
	return true
}

func (c *Connection) CancelQueue() bool {
	c.Lock()
	defer c.Unlock()

	if c.cancelQueue == nil {
		return false
	}

	c.cancelQueue()
	c.cancelQueue = nil
	return true
}

func (c *Connection) InQueue() bool {
	c.Lock()
	hasCancelQueue := c.cancelQueue != nil
	c.Unlock()

	return hasCancelQueue
}

func (c *Connection) Variables() map[string]string {
	return c.variables.Data()
}

// fixme
func test() {
	a := func(c model.Call) {}
	a(&Connection{})
}
