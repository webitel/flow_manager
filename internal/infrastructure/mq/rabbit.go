package mq

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	amqp "github.com/rabbitmq/amqp091-go"

	"github.com/webitel/wlog"

	"github.com/webitel/flow_manager/internal/domain/call"
	"github.com/webitel/flow_manager/internal/domain/chat"
	"github.com/webitel/flow_manager/internal/domain/flow"
	"github.com/webitel/flow_manager/internal/domain/shared/ports"
	"github.com/webitel/flow_manager/internal/infrastructure/pubsub"
	"github.com/webitel/flow_manager/internal/infrastructure/utils"
)

const bufSize = 100

// RabbitEventBus implements ports.EventBus backed by RabbitMQ.
// Publishing uses a confirm-mode channel managed by pubsub.Manager.
// Consuming uses separate per-connection channels created on each reconnect.
type RabbitEventBus struct {
	mgr      *pubsub.Manager
	nodeName string
	log      *wlog.Logger

	callEvent chan call.CallActionData
	execEvent chan flow.ChannelExec
	imEvents  chan chat.MessageWrapper
	ccEvents  chan chat.CCQueueEvent
}

func NewRabbitEventBus(log *wlog.Logger, url, nodeName string) (ports.EventBus, error) {
	r := &RabbitEventBus{
		nodeName:  nodeName,
		log:       log,
		callEvent: make(chan call.CallActionData, bufSize),
		execEvent: make(chan flow.ChannelExec, bufSize),
		imEvents:  make(chan chat.MessageWrapper, bufSize),
		ccEvents:  make(chan chat.CCQueueEvent, bufSize),
	}

	mgr, err := pubsub.New(log, url,
		r.setupCallConsumer,
		r.setupExecConsumer,
		r.setupIMConsumer,
		r.setupCCConsumer,
	)
	if err != nil {
		return nil, err
	}
	r.mgr = mgr
	return r, nil
}

func (r *RabbitEventBus) Publish(ctx context.Context, exchange, key string, data []byte) error {
	return r.mgr.Publish(ctx, exchange, key, data)
}

func (r *RabbitEventBus) Close() {
	r.mgr.Shutdown()
}

func (r *RabbitEventBus) Start() error {
	return r.mgr.Start()
}

func (r *RabbitEventBus) ConsumeCallEvent() <-chan call.CallActionData { return r.callEvent }
func (r *RabbitEventBus) ConsumeExec() <-chan flow.ChannelExec         { return r.execEvent }
func (r *RabbitEventBus) ConsumeIM() <-chan chat.MessageWrapper        { return r.imEvents }
func (r *RabbitEventBus) ConsumeCCEvents() <-chan chat.CCQueueEvent    { return r.ccEvents }

// newConsumerChannel opens a plain amqp channel suitable for consuming.
func newConsumerChannel(conn *amqp.Connection) (*amqp.Channel, error) {
	ch, err := conn.Channel()
	if err != nil {
		return nil, err
	}
	return ch, nil
}

func (r *RabbitEventBus) setupCallConsumer(conn *amqp.Connection, _ *pubsub.Channel) error {
	ch, err := newConsumerChannel(conn)
	if err != nil {
		return err
	}

	if err = ch.ExchangeDeclare(call.FlowExchange, "direct", true, false, false, false, nil); err != nil {
		return fmt.Errorf("declare %s exchange: %w", call.FlowExchange, err)
	}

	if _, err = ch.QueueDeclare(call.CallEventQueueName, true, false, false, false,
		amqp.Table{"x-queue-type": "quorum"}); err != nil {
		return fmt.Errorf("declare queue %s: %w", call.CallEventQueueName, err)
	}

	if err = ch.QueueBind(call.CallEventQueueName, "events.#", call.CallExchange, true, nil); err != nil {
		return fmt.Errorf("bind %s → %s: %w", call.CallEventQueueName, call.CallExchange, err)
	}
	if err = ch.QueueBind(call.CallEventQueueName, "sip.stats", call.OpensipsExchange, true, nil); err != nil {
		return fmt.Errorf("bind %s → %s: %w", call.CallEventQueueName, call.OpensipsExchange, err)
	}

	msgs, err := ch.Consume(call.CallEventQueueName, r.nodeName, false, false, false, false, nil)
	if err != nil {
		return fmt.Errorf("consume %s: %w", call.CallEventQueueName, err)
	}

	go func() {
		for m := range msgs {
			r.log.Debug(fmt.Sprintf("received a message: %s", m.RoutingKey))
			switch m.Exchange {
			case call.CallExchange:
				r.handleCallMessage(m.Body)
			case call.OpensipsExchange:
				r.handleCallMediaStats(m.Body)
			default:
				r.log.Warn(fmt.Sprintf("call consumer: unknown exchange %s", m.Exchange))
			}
			m.Ack(false)
		}
	}()

	return nil
}

func (r *RabbitEventBus) setupExecConsumer(conn *amqp.Connection, _ *pubsub.Channel) error {
	ch, err := newConsumerChannel(conn)
	if err != nil {
		return err
	}

	if err = ch.ExchangeDeclare(call.FlowExchange, "direct", true, false, false, false, nil); err != nil {
		return fmt.Errorf("declare %s exchange: %w", call.FlowExchange, err)
	}

	if _, err = ch.QueueDeclare(call.FlowExecQueueName, true, false, false, false,
		amqp.Table{"x-queue-type": "quorum"}); err != nil {
		return fmt.Errorf("declare queue %s: %w", call.FlowExecQueueName, err)
	}

	if err = ch.QueueBind(call.FlowExecQueueName, "exec", call.FlowExchange, true, nil); err != nil {
		return fmt.Errorf("bind %s: %w", call.FlowExecQueueName, err)
	}

	msgs, err := ch.Consume(call.FlowExecQueueName, "", false, false, false, false, nil)
	if err != nil {
		return fmt.Errorf("consume %s: %w", call.FlowExecQueueName, err)
	}

	go func() {
		for m := range msgs {
			if m.ContentType != "text/json" {
				r.log.Warn(fmt.Sprintf("exec consumer: unexpected content-type %s", m.ContentType))
				m.Ack(false)
				continue
			}
			var data flow.ChannelExec
			if err := json.Unmarshal(m.Body, &data); err != nil {
				r.log.Warn(fmt.Sprintf("exec consumer: parse error: %s", err))
			} else {
				r.execEvent <- data
			}
			m.Ack(false)
		}
	}()

	return nil
}

func (r *RabbitEventBus) setupIMConsumer(conn *amqp.Connection, _ *pubsub.Channel) error {
	ch, err := newConsumerChannel(conn)
	if err != nil {
		return err
	}

	queueName := fmt.Sprintf("%s.%s.any", call.IMQueueNamePrefix, utils.NewId()[0:8])

	if _, err = ch.QueueDeclare(queueName, true, false, false, true,
		amqp.Table{"x-queue-type": "quorum", "x-expires": 10000}); err != nil {
		return fmt.Errorf("declare IM queue: %w", err)
	}

	if err = ch.QueueBind(queueName, "#", call.IMExchange, true, nil); err != nil {
		return fmt.Errorf("bind IM queue: %w", err)
	}

	msgs, err := ch.Consume(queueName, "", false, false, false, false, nil)
	if err != nil {
		return fmt.Errorf("consume IM queue: %w", err)
	}

	go func() {
		for m := range msgs {
			if m.Exchange != call.IMExchange {
				r.log.Warn(fmt.Sprintf("IM consumer: unknown exchange %s", m.Exchange))
				m.Ack(false)
				continue
			}
			var data chat.MessageWrapper
			if err := json.Unmarshal(m.Body, &data); err != nil {
				r.log.Warn(fmt.Sprintf("IM consumer: parse error: %s", err))
				m.Ack(false)
				continue
			}
			if data.Echo {
				m.Ack(false)
				continue
			}
			r.imEvents <- data
			m.Ack(false)
		}
	}()

	return nil
}

func (r *RabbitEventBus) setupCCConsumer(conn *amqp.Connection, _ *pubsub.Channel) error {
	ch, err := newConsumerChannel(conn)
	if err != nil {
		return err
	}

	if err = ch.ExchangeDeclare(call.CallCenterExchange, "topic", true, false, false, false, nil); err != nil {
		return fmt.Errorf("declare %s exchange: %w", call.CallCenterExchange, err)
	}

	queueName := fmt.Sprintf("%s.%s", call.CallCenterPrefix, utils.NewId()[0:8])

	if _, err = ch.QueueDeclare(queueName, true, false, false, true,
		amqp.Table{"x-queue-type": "quorum", "x-expires": 10000}); err != nil {
		return fmt.Errorf("declare CC queue: %w", err)
	}

	if err = ch.QueueBind(queueName, "queue", call.CallCenterExchange, true, nil); err != nil {
		return fmt.Errorf("bind CC queue: %w", err)
	}

	msgs, err := ch.Consume(queueName, "", false, false, false, false, nil)
	if err != nil {
		return fmt.Errorf("consume CC queue: %w", err)
	}

	go func() {
		for m := range msgs {
			if m.Exchange != call.CallCenterExchange {
				r.log.Warn(fmt.Sprintf("CC consumer: unknown exchange %s", m.Exchange))
				m.Ack(false)
				continue
			}
			var ev chat.CCQueueEvent
			if err := json.Unmarshal(m.Body, &ev); err != nil {
				r.log.Warn(fmt.Sprintf("CC consumer: parse error: %s", err))
			} else {
				r.ccEvents <- ev
			}
			m.Ack(false)
		}
	}()

	return nil
}

type jsonRPCCallStats struct {
	Jsonrpc string `json:"jsonrpc"`
	Method  string `json:"method"`
	Params  struct {
		Stats string `json:"stats"`
	} `json:"params"`
}

func (r *RabbitEventBus) handleCallMessage(data []byte) {
	var action call.CallActionData
	if err := json.Unmarshal(data, &action); err != nil {
		r.log.Error(fmt.Sprintf("call consumer: parse error: %s", err))
		return
	}
	r.log.Debug(fmt.Sprintf("call %s [%s]", action.Id, action.Event))
	r.callEvent <- action
}

func (r *RabbitEventBus) handleCallMediaStats(data []byte) {
	var rpc jsonRPCCallStats
	if err := json.Unmarshal(data, &rpc); err != nil {
		r.log.Error(fmt.Sprintf("call stats: parse rpc error: %s", err))
		return
	}

	var callStats call.CallActionMediaStats
	if err := json.Unmarshal([]byte(rpc.Params.Stats), &callStats); err != nil {
		r.log.Error(fmt.Sprintf("call stats: parse stats error: %s", err))
		return
	}

	userId := 0
	if callStats.UserId != nil {
		userId = int(*callStats.UserId)
	}

	callStats.Id = callStats.CallMediaStats.SipId
	callStats.AppId = call.OpensipsExchange
	callStats.Event = call.CallActionStatsName
	callStats.Timestamp = utils.GetMillis()
	if callStats.RTP.RoundTrip.Average > 0 {
		callStats.RTP.RoundTrip.Average /= 1000
		callStats.RTP.RoundTrip.Max /= 1000
		callStats.RTP.RoundTrip.Min /= 1000
	}
	if callStats.RTP.Mos.Average > 0 {
		callStats.RTP.Mos.Average /= 10
		callStats.RTP.Mos.Min /= 10
		callStats.RTP.Mos.Max /= 10
	}

	ca := call.CallActionDataWithUser{
		CallActionData: call.CallActionData{
			CallAction: callStats.CallAction,
		},
	}
	if userId != 0 {
		ca.UserId = strconv.Itoa(userId)
	}

	bodyStats, err := json.Marshal(callStats.CallMediaStats)
	if err != nil {
		r.log.Error(fmt.Sprintf("call stats: marshal error: %s", err))
		return
	}
	strStats := string(bodyStats)
	ca.Data = &strStats

	body, err := json.Marshal(ca)
	if err != nil {
		r.log.Error(fmt.Sprintf("call stats: marshal ca error: %s", err))
		return
	}

	if err = r.mgr.Publish(context.Background(), call.CallExchange,
		fmt.Sprintf("events.stats..%d.%d", callStats.DomainId, userId), body); err != nil {
		r.log.Error(fmt.Sprintf("call stats: publish error: %s", err))
	}
}
