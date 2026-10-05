package emailevent

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"

	"github.com/webitel/wlog"

	"github.com/webitel/flow_manager/model"
)

const (
	retryInitialInterval = time.Second
	retryMaxInterval     = 30 * time.Second
)

// inboxWriter is the part of the Inbox store the consumer needs.
type inboxWriter interface {
	Insert(ctx context.Context, e *model.EmailEvent) (bool, *model.AppError)
}

// consumer moves events from RabbitMQ to the Inbox; a delivery is acknowledged
// only after its row is committed.
type consumer struct {
	url           string
	name          string
	tag           string
	prefetch      int
	deliveryLimit int
	inbox         inboxWriter
	log           *wlog.Logger
	minBackoff    time.Duration
	maxBackoff    time.Duration
}

func newConsumer(url, nodeId, runId string, cfg model.EmailEventsSettings, inbox inboxWriter, log *wlog.Logger) *consumer {
	return &consumer{
		url:           url,
		name:          "flow_manager email events " + nodeId,
		tag:           nodeId + "/" + runId,
		prefetch:      cfg.Prefetch,
		deliveryLimit: cfg.DeliveryLimit,
		inbox:         inbox,
		log:           log.With(wlog.String("component", "email_events_consumer")),
		minBackoff:    retryInitialInterval,
		maxBackoff:    retryMaxInterval,
	}
}

// Run reconnects until ctx is done; an unavailable broker never fails it.
func (c *consumer) Run(ctx context.Context) {
	var backoff time.Duration

	for ctx.Err() == nil {
		consumed, err := c.generation(ctx)
		if ctx.Err() != nil {
			break
		}
		if consumed {
			backoff = 0
		}

		backoff = c.next(backoff)
		c.log.Warn("email events consumer paused", wlog.Duration("retry_in", backoff), wlog.Err(err))
		sleep(ctx, backoff)
	}

	c.log.Info("email events consumer stopped")
}

// generation serves one connection and channel; consumed reports it reached
// Consume. Unacknowledged deliveries return to the queue when it closes.
func (c *consumer) generation(ctx context.Context) (consumed bool, err error) {
	props := amqp.NewConnectionProperties()
	props.SetClientConnectionName(c.name)

	conn, err := amqp.DialConfig(c.url, amqp.Config{Locale: "en_US", Properties: props})
	if err != nil {
		return false, fmt.Errorf("connect: %w", err)
	}
	defer conn.Close()

	ch, err := conn.Channel()
	if err != nil {
		return false, fmt.Errorf("open channel: %w", err)
	}
	defer ch.Close()

	if err := ch.Qos(c.prefetch, 0, false); err != nil {
		return false, fmt.Errorf("set prefetch: %w", err)
	}
	if err := declare(ch, c.deliveryLimit); err != nil {
		return false, err
	}

	deliveries, err := ch.Consume(InboundQueue, c.tag, false, false, false, false, nil)
	if err != nil {
		return false, fmt.Errorf("consume: %w", err)
	}

	// Cancelled first among the defers, so a retrying insert stops before Close.
	genCtx, cancel := context.WithCancelCause(ctx)
	defer cancel(nil)

	connClosed := conn.NotifyClose(make(chan *amqp.Error, 1))
	chClosed := ch.NotifyClose(make(chan *amqp.Error, 1))
	go func() {
		select {
		case e := <-connClosed:
			cancel(fmt.Errorf("connection closed: %v", e))
		case e := <-chClosed:
			cancel(fmt.Errorf("channel closed: %v", e))
		case <-genCtx.Done():
		}
	}()

	c.log.Info("email events consumer started", wlog.String("queue", InboundQueue), wlog.String("consumer_tag", c.tag))

	for {
		select {
		case <-genCtx.Done():
			return true, context.Cause(genCtx)
		case d, ok := <-deliveries:
			if !ok {
				return true, errors.New("deliveries closed")
			}
			if err := c.handle(genCtx, d); err != nil {
				return true, err
			}
		}
	}
}

// handle settles one delivery; an error ends the generation. A cancelled ctx
// leaves the delivery unsettled for redelivery.
func (c *consumer) handle(ctx context.Context, d amqp.Delivery) error {
	headerId := headerString(d.Headers[HeaderEventId])
	log := c.log.With(wlog.String("email_event_id", headerId))

	event, err := Decode(d.Body)
	if err == nil && headerId != "" && headerId != event.EventId {
		err = fmt.Errorf("header %s %q differs from event_id %q", HeaderEventId, headerId, event.EventId)
	}
	if err != nil {
		return c.reject(d, log, "invalid_event", err)
	}

	row := &model.EmailEvent{
		EventId:       event.EventId,
		DomainId:      event.DomainId,
		EventType:     event.EventType,
		SchemaVersion: event.SchemaVersion,
		ThreadId:      event.ThreadId,
		MessageId:     event.MessageId,
		FlowId:        event.FlowId,
		Payload:       d.Body,
	}
	log = c.log.With(
		wlog.String("email_event_id", event.EventId),
		wlog.Int64("domain_id", event.DomainId),
		wlog.Int64("email_thread_id", event.ThreadId),
		wlog.Int64("email_message_id", event.MessageId),
	)

	var backoff time.Duration
	for {
		inserted, appErr := c.inbox.Insert(ctx, row)
		switch {
		case appErr == nil:
			if inserted {
				log.Info("email event stored in the inbox")
			} else {
				log.Debug("email event already in the inbox")
			}
			if err := d.Ack(false); err != nil {
				return fmt.Errorf("ack: %w", err)
			}

			return nil
		case appErr.StatusCode == http.StatusBadRequest:
			return c.reject(d, log, "invalid_data", appErr)
		case ctx.Err() != nil:
			return nil
		}

		backoff = c.next(backoff)
		log.Warn("email event inbox write failed", wlog.Duration("retry_in", backoff), wlog.Err(appErr))
		if !sleep(ctx, backoff) {
			return nil
		}
	}
}

// reject sends a delivery that no retry can store to the dead letter queue.
func (c *consumer) reject(d amqp.Delivery, log *wlog.Logger, category string, reason error) error {
	log.Warn("email event sent to the dead letter queue",
		wlog.String("category", category), wlog.String("routing_key", d.RoutingKey), wlog.Err(reason))

	if err := d.Nack(false, false); err != nil {
		return fmt.Errorf("nack: %w", err)
	}

	return nil
}

func (c *consumer) next(backoff time.Duration) time.Duration {
	return nextBackoff(backoff, c.minBackoff, c.maxBackoff)
}

// nextBackoff doubles the previous pause within [lo, hi].
func nextBackoff(prev, lo, hi time.Duration) time.Duration {
	return min(max(2*prev, lo), hi)
}

func headerString(v any) string {
	switch v := v.(type) {
	case string:
		return v
	case []byte:
		return string(v)
	default:
		return ""
	}
}

// sleep waits for d and reports whether ctx is still running.
func sleep(ctx context.Context, d time.Duration) bool {
	timer := time.NewTimer(d)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}
