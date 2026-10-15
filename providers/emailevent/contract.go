// Package emailevent consumes email-service events into the flow.email_event
// Inbox and starts Flows from them through the email router.
package emailevent

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/pborman/uuid"

	"github.com/webitel/flow_manager/model"
)

// Contract of email-service, described in its docs/events.md.
const (
	TypeInboundMessage       = "email.message.inbound"
	InboundMessageSchema     = 1
	InboundMessageRoutingKey = "email.message.inbound.v1"

	kindRegular = "regular"
)

// InboundMessage is the event of a new inbound Message; unknown fields are ignored.
type InboundMessage struct {
	EventId       string    `json:"event_id"`
	EventType     string    `json:"event_type"`
	SchemaVersion int32     `json:"schema_version"`
	OccurredAt    time.Time `json:"occurred_at"`
	DomainId      int64     `json:"domain_id"`
	ProfileId     int64     `json:"profile_id"`
	FlowId        *int64    `json:"flow_id"`
	ThreadId      int64     `json:"thread_id"`
	MessageId     int64     `json:"message_id"`
	Snapshot      Snapshot  `json:"snapshot"`
}

// Snapshot is the Message as it was when it became ready, already truncated.
type Snapshot struct {
	InternetMessageId string       `json:"internet_message_id"`
	InReplyTo         string       `json:"in_reply_to"`
	Subject           string       `json:"subject"`
	MessageKind       string       `json:"message_kind"`
	ThreadKind        string       `json:"thread_kind"`
	ContactId         *int64       `json:"contact_id"`
	ReceivedAt        time.Time    `json:"received_at"`
	From              []Address    `json:"from"`
	Sender            []Address    `json:"sender"`
	ReplyTo           []Address    `json:"reply_to"`
	To                []Address    `json:"to"`
	Cc                []Address    `json:"cc"`
	BodyText          string       `json:"body_text"`
	Attachments       []model.File `json:"attachments"`
	BodyTextTruncated bool         `json:"body_text_truncated"`
}

type Address struct {
	Address string `json:"address"`
	Name    string `json:"name"`
}

// Decode parses and validates an event body.
func Decode(body []byte) (*InboundMessage, error) {
	var m InboundMessage
	if err := json.Unmarshal(body, &m); err != nil {
		return nil, fmt.Errorf("decode email event: %w", err)
	}
	if err := m.Validate(); err != nil {
		return nil, err
	}

	return &m, nil
}

// Validate also guarantees the event passes every check of flow.email_event.
func (m *InboundMessage) Validate() error {
	var errs []error
	check := func(ok bool, format string, args ...any) {
		if !ok {
			errs = append(errs, fmt.Errorf(format, args...))
		}
	}

	check(m.EventType == TypeInboundMessage, "unsupported event_type %q", m.EventType)
	check(m.SchemaVersion == InboundMessageSchema, "unsupported schema_version %d", m.SchemaVersion)
	// Only the canonical form, which PostgreSQL accepts for uuid.
	check(len(m.EventId) == 36 && uuid.Parse(m.EventId) != nil, "invalid event_id %q", m.EventId)
	check(m.DomainId > 0, "invalid domain_id %d", m.DomainId)
	check(m.ProfileId > 0, "invalid profile_id %d", m.ProfileId)
	check(m.ThreadId > 0, "invalid thread_id %d", m.ThreadId)
	check(m.MessageId > 0, "invalid message_id %d", m.MessageId)
	check(m.FlowId == nil || *m.FlowId > 0 && *m.FlowId <= math.MaxInt, "invalid flow_id %d", deref(m.FlowId))
	check(m.Snapshot.MessageKind == kindRegular, "unsupported message_kind %q", m.Snapshot.MessageKind)
	check(m.Snapshot.ThreadKind == kindRegular, "unsupported thread_kind %q", m.Snapshot.ThreadKind)

	if err := errors.Join(errs...); err != nil {
		return fmt.Errorf("invalid email event: %w", err)
	}

	return nil
}

func deref(v *int64) int64 {
	if v == nil {
		return 0
	}

	return *v
}
