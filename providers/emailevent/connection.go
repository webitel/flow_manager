package emailevent

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"net/http"
	"strconv"
	"strings"
	"sync"

	"github.com/tidwall/gjson"

	"github.com/webitel/wlog"

	"github.com/webitel/flow_manager/model"
)

type launchState int

const (
	launchPending launchState = iota
	launchStarted
	launchAbandoned
)

// Connection hands one Inbox event to the email router; the worker that claimed
// the event waits for its launch result.
type Connection struct {
	mu                 sync.RWMutex
	event              *InboundMessage
	initialFromAddress string
	nodeId             string
	schemaId           int
	variables          model.Variables
	state              launchState
	launched           chan *model.AppError
	ctx                context.Context //nolint:containedctx
	cancel             context.CancelFunc
	log                *wlog.Logger
}

var (
	_ model.EmailLaunchConnection = (*Connection)(nil)
	_ model.EmailEventConnection  = (*Connection)(nil)
)

// NewConnection needs an event with flow_id. ctx belongs to the server, so the
// Flow outlives the worker and its handshake.
func NewConnection(ctx context.Context, nodeId string, event *InboundMessage, log *wlog.Logger) *Connection {
	ctx, cancel := context.WithCancel(ctx)
	var from string
	if !event.Snapshot.FromTruncated && len(event.Snapshot.From) != 0 {
		from = model.NormalizeEmailRoutingAddress(event.Snapshot.From[0].Address)
	}

	return &Connection{
		event:              event,
		initialFromAddress: from,
		nodeId:             nodeId,
		schemaId:           int(*event.FlowId),
		variables:          variables(event),
		launched:           make(chan *model.AppError, 1),
		ctx:                ctx,
		cancel:             cancel,
		log: log.With(
			wlog.Namespace("context"),
			wlog.String("email_event_id", event.EventId),
			wlog.Int64("email_thread_id", event.ThreadId),
			wlog.Int64("email_message_id", event.MessageId),
		),
	}
}

// variables keeps the names and formats of the IMAP connection, so existing
// schemas read them unchanged.
func variables(event *InboundMessage) model.Variables {
	s := event.Snapshot
	vars := model.Variables{
		"id":          strconv.FormatInt(event.MessageId, 10),
		"message_id":  s.InternetMessageId,
		"from":        addresses(s.From),
		"to":          addresses(s.To),
		"cc":          addresses(s.Cc),
		"sender":      addresses(s.Sender),
		"reply_to":    addresses(s.ReplyTo),
		"in_reply_to": s.InReplyTo,
		"subject":     s.Subject,
		"body":        s.BodyText,

		"email_event_id":   event.EventId,
		"email_profile_id": strconv.FormatInt(event.ProfileId, 10),
		"email_thread_id":  strconv.FormatInt(event.ThreadId, 10),
		"email_message_id": strconv.FormatInt(event.MessageId, 10),
		"body_truncated":   strconv.FormatBool(s.BodyTextTruncated),
	}

	if len(s.Attachments) != 0 {
		if attachments, err := json.Marshal(s.Attachments); err == nil {
			vars["attachments"] = string(attachments)
		}
	}
	if s.ContactId != nil {
		vars["wbt_contact_id"] = strconv.FormatInt(*s.ContactId, 10)
	}

	return vars
}

func addresses(list []Address) string {
	values := make([]string, 0, len(list))
	for _, a := range list {
		values = append(values, a.Address)
	}

	return strings.Join(values, ",")
}

// MarkStarted moves a pending launch to started; false once it was abandoned.
func (c *Connection) MarkStarted() bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.state != launchPending {
		return c.state == launchStarted
	}
	c.state = launchStarted

	return true
}

// Abandon moves a pending launch to abandoned; false once the Flow started.
func (c *Connection) Abandon() bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.state != launchPending {
		return c.state == launchAbandoned
	}
	c.state = launchAbandoned

	return true
}

// ReportLaunch never blocks: only the first result is kept.
func (c *Connection) ReportLaunch(err *model.AppError) {
	select {
	case c.launched <- err:
	default:
	}
}

// LaunchResult delivers the result of ReportLaunch.
func (c *Connection) LaunchResult() <-chan *model.AppError {
	return c.launched
}

func (c *Connection) Type() model.ConnectionType {
	return model.ConnectionTypeEmail
}

func (c *Connection) Id() string {
	return c.event.EventId
}

func (c *Connection) NodeId() string {
	return c.nodeId
}

func (c *Connection) DomainId() int64 {
	return c.event.DomainId
}

func (c *Connection) EventId() string {
	return c.event.EventId
}

func (c *Connection) ThreadId() int64 {
	return c.event.ThreadId
}

func (c *Connection) MessageId() int64 {
	return c.event.MessageId
}

func (c *Connection) InitialFromAddress() string {
	return c.initialFromAddress
}

func (c *Connection) SchemaId() int {
	return c.schemaId
}

func (c *Connection) Context() context.Context {
	return c.ctx
}

func (c *Connection) Log() *wlog.Logger {
	return c.log
}

// Email is nil: the event has no cc_email row behind it.
func (c *Connection) Email() *model.Email {
	return nil
}

func (c *Connection) Reply(string) (*model.Email, *model.AppError) {
	return nil, model.NewAppError("EmailEvent.Reply", "email_event.reply.not_supported", nil,
		"reply is not supported for email-service events", http.StatusNotImplemented)
}

func (c *Connection) Get(name string) (string, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if idx := strings.Index(name, "."); idx > 0 {
		if v, ok := c.variables[name[:idx]]; ok {
			return gjson.Get(fmt.Sprintf("%v", v), name[idx+1:]).String(), true
		}
	}

	v, ok := c.variables[name]
	if !ok {
		return "", false
	}

	return fmt.Sprintf("%v", v), ok
}

func (c *Connection) Set(_ context.Context, vars model.Variables) (model.Response, *model.AppError) {
	c.mu.Lock()
	defer c.mu.Unlock()

	maps.Copy(c.variables, vars)

	return model.CallResponseOK, nil
}

func (c *Connection) Variables() map[string]string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	vars := make(map[string]string, len(c.variables))
	for k, v := range c.variables {
		vars[k] = fmt.Sprintf("%v", v)
	}

	return vars
}

func (c *Connection) ParseText(text string, ops ...model.ParseOption) string {
	return model.ParseText(c, text, ops...)
}

// Close cancels the Flow; a launch that has not started is abandoned.
func (c *Connection) Close() *model.AppError {
	c.Abandon()
	c.cancel()

	return nil
}
