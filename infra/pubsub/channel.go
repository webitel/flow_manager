package pubsub

import (
	"context"
	"errors"
	"sync"

	"github.com/google/uuid"
	amqp "github.com/rabbitmq/amqp091-go"
)

type MQExchangeType string

const (
	ExchangeTypeFanout MQExchangeType = "fanout"
	ExchangeTypeTopic  MQExchangeType = "topic"
	ExchangeTypeDirect MQExchangeType = "direct"
)

type Headers amqp.Table
type Delivery <-chan amqp.Delivery

// Exchange is the rabbitmq exchange.
type Exchange struct {
	Name    string
	Type    MQExchangeType
	Durable bool
}

type Channel struct {
	uuid           string
	connection     *amqp.Connection
	channel        *amqp.Channel
	confirmPublish chan amqp.Confirmation
	mtx            sync.Mutex
}

func newChannel(conn *amqp.Connection, prefetchCount int, prefetchGlobal bool, confirmPublish bool) (*Channel, error) {
	id, err := uuid.NewRandom()
	if err != nil {
		return nil, err
	}

	rabbitCh := &Channel{
		uuid:       id.String(),
		connection: conn,
	}

	if err := rabbitCh.Connect(prefetchCount, prefetchGlobal, confirmPublish); err != nil {
		return nil, err
	}

	return rabbitCh, nil
}

func (r *Channel) Connect(prefetchCount int, prefetchGlobal bool, confirmPublish bool) error {
	var err error

	r.channel, err = r.connection.Channel()
	if err != nil {
		return err
	}

	if err = r.channel.Qos(prefetchCount, 0, prefetchGlobal); err != nil {
		return err
	}

	if confirmPublish {
		r.confirmPublish = r.channel.NotifyPublish(make(chan amqp.Confirmation, 1))
		if err = r.channel.Confirm(false); err != nil {
			return err
		}
	}

	return nil
}

func (r *Channel) Close() error {
	if r.channel == nil {
		return errors.New("channel is nil")
	}

	return r.channel.Close()
}

func (r *Channel) Publish(ctx context.Context, exchange, key string, body []byte) error {
	if r.channel == nil {
		return errors.New("channel is nil")
	}

	if r.confirmPublish != nil {
		r.mtx.Lock()
		defer r.mtx.Unlock()
	}

	message := amqp.Publishing{
		ContentType: "text/json",
		Body:        body,
	}

	if err := r.channel.PublishWithContext(ctx, exchange, key, false, false, message); err != nil {
		return err
	}

	if r.confirmPublish != nil {
		confirmation, ok := <-r.confirmPublish
		if !ok {
			return errors.New("channel closed before could receive confirmation of publish")
		}

		if !confirmation.Ack {
			return errors.New("could not publish message, received nack from broker on confirmation")
		}
	}

	return nil
}

func (r *Channel) DeclareExchange(ex Exchange) error {
	return r.channel.ExchangeDeclare(
		ex.Name,         // name
		string(ex.Type), // kind
		ex.Durable,      // durable
		false,           // autoDelete
		false,           // internal
		false,           // noWait
		nil,             // args
	)
}

func (r *Channel) DeclareDurableExchange(ex Exchange) error {
	return r.channel.ExchangeDeclare(
		ex.Name,         // name
		string(ex.Type), // kind
		true,            // durable
		false,           // autoDelete
		false,           // internal
		false,           // noWait
		nil,             // args
	)
}

func (r *Channel) DeclareQueue(queue string, args Headers) error {
	_, err := r.channel.QueueDeclare(
		queue,            // name
		false,            // durable
		true,             // autoDelete
		false,            // exclusive
		false,            // noWait
		amqp.Table(args), // args
	)
	return err
}

func (r *Channel) DeclareDurableQueue(queue string, args Headers) error {
	_, err := r.channel.QueueDeclare(
		queue,            // name
		true,             // durable
		false,            // autoDelete
		false,            // exclusive
		false,            // noWait
		amqp.Table(args), // args
	)
	return err
}

func (r *Channel) DeclareReplyQueue(queue string) error {
	_, err := r.channel.QueueDeclare(
		queue, // name
		false, // durable
		true,  // autoDelete
		true,  // exclusive
		false, // noWait
		nil,   // args
	)
	return err
}

func (r *Channel) ConsumeQueue(queue string, autoAck bool) (Delivery, error) {
	return r.channel.Consume(
		queue,   // queue
		r.uuid,  // consumer
		autoAck, // autoAck
		false,   // exclusive
		false,   // nolocal
		false,   // nowait
		nil,     // args
	)
}

func (r *Channel) BindQueue(queue, key, exchange string, args Headers) error {
	return r.channel.QueueBind(
		queue,            // name
		key,              // key
		exchange,         // exchange
		false,            // noWait
		amqp.Table(args), // args
	)
}
