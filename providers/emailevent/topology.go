package emailevent

import (
	"fmt"

	amqp "github.com/rabbitmq/amqp091-go"
)

// Topology of email-service events; every object is durable and its parameters
// match docs/events.md of email-service, since a redeclare with others fails.
const (
	Exchange                = "email.events"
	InboundQueue            = "flow.email.inbound"
	DeadLetterExchange      = "email.events.dlx"
	InboundDeadLetterQueue  = "flow.email.inbound.dlq"
	HeaderEventId           = "x-event-id"
	exchangeTypeTopic       = "topic"
	exchangeTypeFanout      = "fanout"
	queueTypeQuorum         = "quorum"
	deadLetterAnyRoutingKey = ""
)

// declare creates the topology; deliveryLimit moves an event that keeps coming
// back unacknowledged to the dead letter queue.
func declare(ch *amqp.Channel, deliveryLimit int) error {
	for _, exchange := range []struct{ name, kind string }{
		{Exchange, exchangeTypeTopic},
		{DeadLetterExchange, exchangeTypeFanout},
	} {
		if err := ch.ExchangeDeclare(exchange.name, exchange.kind, true, false, false, false, nil); err != nil {
			return fmt.Errorf("declare exchange %s: %w", exchange.name, err)
		}
	}

	queues := []struct {
		name, exchange, key string
		args                amqp.Table
	}{
		{InboundQueue, Exchange, InboundMessageRoutingKey, amqp.Table{
			"x-queue-type":           queueTypeQuorum,
			"x-dead-letter-exchange": DeadLetterExchange,
			"x-delivery-limit":       deliveryLimit,
		}},
		{InboundDeadLetterQueue, DeadLetterExchange, deadLetterAnyRoutingKey, amqp.Table{
			"x-queue-type": queueTypeQuorum,
		}},
	}
	for _, q := range queues {
		if _, err := ch.QueueDeclare(q.name, true, false, false, false, q.args); err != nil {
			return fmt.Errorf("declare queue %s: %w", q.name, err)
		}
		if err := ch.QueueBind(q.name, q.key, q.exchange, false, nil); err != nil {
			return fmt.Errorf("bind queue %s: %w", q.name, err)
		}
	}

	return nil
}
