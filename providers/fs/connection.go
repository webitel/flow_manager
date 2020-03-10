package fs

import (
	"bytes"
	"context"
	"fmt"
	uuid "github.com/satori/go.uuid"
	"github.com/webitel/flow_manager/model"
	"github.com/webitel/flow_manager/providers/fs/eventsocket"
	"github.com/webitel/wlog"
	"net/http"
	"strconv"
	"sync"
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

var errExecuteAfterHangup = model.NewAppError("FreeSWITCH", "provider.fs.execute.after_hangup", nil, "not allow after hangup", http.StatusBadRequest)

type Connection struct {
	id              string
	uuid            string
	nodeId          string
	nodeName        string
	context         string
	destination     string
	stopped         bool
	direction       model.CallDirection
	gatewayId       int
	domainId        int
	domainName      string
	from            *model.CallEndpoint
	to              *model.CallEndpoint
	systemDirection string

	userId           int
	disconnected     chan struct{}
	lastEvent        *eventsocket.Event
	connection       *eventsocket.Connection
	callbackMessages map[string]chan *eventsocket.Event
	variables        map[string]string
	hangupCause      string
	sync.RWMutex
}

func (c Connection) Type() model.ConnectionType {
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
	connection := &Connection{
		uuid:             dump.Get(HEADER_ID_NAME),
		nodeId:           dump.Get(HEADER_CORE_ID_NAME),
		nodeName:         dump.Get(HEADER_CORE_NAME),
		context:          dump.Get(HEADER_CONTEXT_NAME),
		direction:        getDirection(dump.Get(HEADER_DIRECTION_NAME)),
		gatewayId:        getIntFromStr(dump.Get(HEADER_GATEWAY_ID)),
		domainId:         getIntFromStr(dump.Get(HEADER_DOMAIN_ID)),
		userId:           getIntFromStr(dump.Get(HEADER_USER_ID)),
		connection:       baseConnection,
		lastEvent:        dump,
		callbackMessages: make(map[string]chan *eventsocket.Event),
		disconnected:     make(chan struct{}),
		variables:        make(map[string]string),
	}
	connection.setCallInfo(dump)
	connection.initDestination(dump)
	connection.updateVariablesFromEvent(dump)
	return connection
}

func (c *Connection) setCallInfo(dump *eventsocket.Event) {
	direction := dump.Get("variable_sip_h_X-Webitel-Direction")
	if direction == "internal" {
		if dump.Get("Call-Direction") == "outbound" {
			direction = "inbound"
		} else {
			direction = "outbound"
		}
	}
	c.direction = getDirection(direction)
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
			//FIXME
		} else {
			c.from.Type = model.CallEndpointTypeUser
			c.from.Id = fmt.Sprintf("%d", c.userId)
			c.from.Name = dump.Get("Caller-Caller-ID-Name")
			c.from.Number = dump.Get("Caller-Caller-ID-Number")
		}
		//fmt.Println(direction)
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
	return c.uuid
}

func (c *Connection) DomainId() int {
	return c.domainId
}

func (c *Connection) SetDomainName(name string) {
	c.domainName = name
}

func (c *Connection) DomainName() string {
	return c.domainName
}

func (c *Connection) UserId() int {
	return c.userId
}

func (c *Connection) ParseText(text string) string {
	return text
}

func (c *Connection) Context() string {
	return c.context
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

func (c *Connection) SetDirection(direction string) error {
	if c.direction == "" {
		if _, err := c.Execute(context.Background(), "set", "webitel_direction="+direction); err != nil {
			return err
		}
		c.direction = getDirection(direction)
	}
	return nil
}

func (c *Connection) Get(key string) (value string, ok bool) {
	if c.Stopped() {
		value, ok = c.variables[key]
	} else if c.lastEvent != nil {
		value = c.lastEvent.Get("variable_" + c.UserVariablePrefix(key))
		if value != "" {
			ok = true
		}
	}
	return
}

func (c *Connection) setDisconnectedVariables(vars model.Variables) (model.Response, *model.AppError) {
	for k, v := range vars {
		c.variables[k] = fmt.Sprintf("%v", v)
	}
	return model.CallResponseOK, nil
}

func (c *Connection) setChannelVariables(vars model.Variables) (model.Response, *model.AppError) {
	str := "^^"
	for k, v := range vars {
		str += fmt.Sprintf(`~'%s'='%v'`, k, v)
	}

	return c.Execute(context.Background(), "multiset", str)
}

func (c Connection) UserVariablePrefix(name string) string {
	return fmt.Sprintf("usr_%s", name)
}

func (c *Connection) Set(vars model.Variables) (model.Response, *model.AppError) {
	if len(vars) == 0 {
		return nil, model.NewAppError("Call.Set", "call.app.set.valid.args", nil, "bad request", http.StatusBadRequest)
	}

	if c.Stopped() {
		return c.setDisconnectedVariables(vars)
	} else {
		return c.setChannelVariables(vars)
	}
}

func (c *Connection) SetAll(vars model.Variables) (model.Response, *model.AppError) {
	var err *model.AppError
	for k, v := range vars {
		if _, err = c.Execute(context.Background(), "export", fmt.Sprintf(`'%s'='%v'`, c.UserVariablePrefix(k), v)); err != nil {
			return nil, err
		}
	}

	return model.CallResponseOK, nil
}

func (c *Connection) SetNoLocal(vars model.Variables) (model.Response, *model.AppError) {
	var err *model.AppError
	for k, v := range vars {
		if _, err = c.Execute(context.Background(), "export", fmt.Sprintf(`nolocal:'%s'='%v'`, k, v)); err != nil {
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
			wlog.Debug(fmt.Sprintf("call %s executed app: %s %s %s", c.Id(), event.Get(HEADER_APPLICATION_NAME),
				event.Get(HEADER_APPLICATION_DATA_NAME), event.Get(HEADER_APPLICATION_RESPONSE_NAME)))
		case EVENT_HANGUP_COMPLETE:
			c.hangupCause = event.Get(HEADER_HANGUP_CAUSE_NAME)
			wlog.Debug(fmt.Sprintf("call %s hangup %s", c.Id(), c.hangupCause))
			//TODO SET DISCONNECT ROUTE
			c.connection.Send("exit")
			c.stopped = true
		default:
			wlog.Debug(fmt.Sprintf("call %s receive event %s", c.Id(), event.Get(HEADER_EVENT_NAME)))
		}
	} else if event.Get(HEADER_CONTENT_TYPE_NAME) == "text/disconnect-notice" && event.Get(HEADER_CONTENT_DISPOSITION_NAME) == "Disconnected" {

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
	return c.hangupCause
}

func (c *Connection) Execute(ctx context.Context, app string, args interface{}) (model.Response, *model.AppError) {
	if c.Stopped() {
		return nil, errExecuteAfterHangup
	}

	wlog.Debug(fmt.Sprintf("call %s try execute %s %v", c.uuid, app, args))

	guid := uuid.NewV4()
	var err error

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

	if c.Stopped() {
		return nil, errExecuteAfterHangup
	}

	select {
	case <-ctx.Done():
		return model.CallResponse{Status: "CANCEL"}, model.NewAppError("FreeSWITCH", "provider.fs.execute.cancel", nil, "cancel", http.StatusInternalServerError)
	case <-e:
		return model.CallResponseOK, nil
	}
}

func (c *Connection) updateVariablesFromEvent(event *eventsocket.Event) {
	for k, _ := range event.Header {
		c.variables[k] = event.Get(k)
	}
}

func (c *Connection) GetVariable(name string) (value string) {
	if c.lastEvent != nil {
		value = c.lastEvent.Get(name)
	}

	return
}

func (c *Connection) GetGlobalVariables() (map[string]string, error) {
	variables := make(map[string]string)
	data, err := c.Api("global_getvar")
	if err != nil {
		return variables, err
	}

	rows := bytes.Split(data, []byte("\n"))
	var val [][]byte
	for i := 0; i < len(rows); i++ {
		val = bytes.SplitN(rows[i], []byte("="), 2)
		if len(val) == 2 {
			variables[string(val[0])] = string(val[1])
		}
	}
	return variables, nil
}

func (c *Connection) WaitForDisconnect() {
	<-c.disconnected
}

func (c *Connection) SendEvent(m map[string]string, name string) error {
	return c.connection.SendEvent(m, name)
}

func (c *Connection) DumpVariables() map[string]string {
	return c.variables
}

//fixme
func test() {
	a := func(c model.Call) {}
	a(&Connection{})
}
