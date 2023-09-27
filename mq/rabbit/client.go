package rabbit

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sync"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/webitel/flow_manager/model"
	"github.com/webitel/flow_manager/mq"
	"github.com/webitel/wlog"
)

const CallChanBufferCount = 100

const (
	MAX_ATTEMPTS_CONNECT = 100
	RECONNECT_SEC        = 5
)

const (
	EXIT_DECLARE_QUEUE = 111
	EXIT_BIND          = 112
)

type AMQP struct {
	settings           *model.MQSettings
	connection         *amqp.Connection
	channel            *amqp.Channel
	callName           string
	execName           string
	nodeName           string
	connectionAttempts int
	stopping           bool
	callEvent          chan model.CallActionData
	execEvent          chan model.ChannelExec
	queueEvent         mq.QueueEvent
	sync.RWMutex
}

func NewRabbitMQ(settings model.MQSettings, nodeName string) mq.LayeredMQLayer {
	mq_ := &AMQP{
		settings:  &settings,
		callEvent: make(chan model.CallActionData, CallChanBufferCount),
		execEvent: make(chan model.ChannelExec, CallChanBufferCount),
		nodeName:  nodeName,
	}
	mq_.queueEvent = NewQueueMQ(mq_)
	mq_.initConnection()

	return mq_
}

func (a *AMQP) QueueEvent() mq.QueueEvent {
	return a.queueEvent
}

func (a *AMQP) initConnection() {
	var err error

	if a.connectionAttempts >= MAX_ATTEMPTS_CONNECT {
		wlog.Critical(fmt.Sprintf("Failed to open AMQP connection..."))
		time.Sleep(time.Second)
		os.Exit(1)
	}
	a.connectionAttempts++
	a.connection, err = amqp.Dial(a.settings.Url)
	if err != nil {
		wlog.Critical(fmt.Sprintf("Failed to open AMQP connection to err:%v", err.Error()))
		time.Sleep(time.Second * RECONNECT_SEC)
		a.initConnection()
	} else {
		a.connectionAttempts = 0
		a.Lock()
		a.channel, err = a.connection.Channel()
		a.Unlock()
		if err != nil {
			wlog.Critical(fmt.Sprintf("Failed to open AMQP channel to err:%v", err.Error()))
			time.Sleep(time.Second)
			os.Exit(1)
		} else {
			a.initExchange()
			a.initQueues()
		}
	}
}

func (a *AMQP) initExchange() {
	err := a.channel.ExchangeDeclare(
		model.FlowExchange,
		"direct",
		true,
		false,
		false,
		false,
		nil,
	)

	if err != nil {
		wlog.Critical(fmt.Sprintf("Failed to create AMQP exchange to err:%v", err.Error()))
		time.Sleep(time.Second)
		os.Exit(1)
	}
}

func (a *AMQP) initQueues() {
	var err error
	var callQueue amqp.Queue
	var execQueue amqp.Queue

	callQueue, err = a.channel.QueueDeclare(model.CallEventQueueName, true, false, false,
		true, nil)
	if err != nil {
		wlog.Critical(fmt.Sprintf("Failed to declare AMQP queue %v to err:%v", model.CallEventQueueName, err.Error()))
		time.Sleep(time.Second)
		os.Exit(EXIT_DECLARE_QUEUE)
	}

	execQueue, err = a.channel.QueueDeclare(model.FlowExecQueueName, true, false, false,
		true, nil)
	if err != nil {
		wlog.Critical(fmt.Sprintf("Failed to declare AMQP queue %v to err:%v", model.FlowExecQueueName, err.Error()))
		time.Sleep(time.Second)
		os.Exit(EXIT_DECLARE_QUEUE)
	}

	a.callName = callQueue.Name
	a.execName = execQueue.Name
	wlog.Debug(fmt.Sprintf("Success declare queue %v, %v, connected consumers %v", callQueue.Name, execQueue.Name, callQueue.Consumers))
	a.subscribeCall()
	a.subscribeExec()
}

func (a *AMQP) subscribeCall() {
	err := a.channel.QueueBind(a.callName, "events.#", model.CallExchange, true, nil)
	if err != nil {
		wlog.Critical(fmt.Sprintf("Error binding queue %s to %s: %s", a.callName, model.CallExchange, err.Error()))
		time.Sleep(time.Second)
		os.Exit(EXIT_BIND)
	}

	msgs, err := a.channel.Consume(
		a.callName,
		"",
		false,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		wlog.Critical(fmt.Sprintf("Error create consume for queue %s: %s", a.callName, err.Error()))
		time.Sleep(time.Second)
		os.Exit(EXIT_BIND)
	}

	go func() {

		for m := range msgs {
			if m.ContentType != "text/json" {
				wlog.Warn(fmt.Sprintf("Failed receive event content type: %v\n%s", m.ContentType, m.Body))
				continue
			}

			switch m.Exchange {
			case model.CallExchange:
				a.handleCallMessage(m.Body)
			default:
				wlog.Warn(fmt.Sprintf("unable to parse event, not found exchange %s", m.Exchange))
			}
			m.Ack(false)
		}

		if !a.stopping {
			a.initConnection()
		}
	}()
}

func (a *AMQP) subscribeExec() {
	err := a.channel.QueueBind(a.execName, "exec", model.FlowExchange, true, nil)
	if err != nil {
		wlog.Critical(fmt.Sprintf("Error binding queue %s to %s: %s", a.execName, model.FlowExchange, err.Error()))
		time.Sleep(time.Second)
		os.Exit(EXIT_BIND)
	}

	msgs, err := a.channel.Consume(
		a.execName,
		"",
		false,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		wlog.Critical(fmt.Sprintf("Error create consume for queue %s: %s", a.execName, err.Error()))
		time.Sleep(time.Second)
		os.Exit(EXIT_BIND)
	}

	go func() {

		for m := range msgs {
			if m.ContentType != "text/json" {
				wlog.Warn(fmt.Sprintf("Failed receive event content type: %v\n%s", m.ContentType, m.Body))
				continue
			}

			switch m.Exchange {
			case model.FlowExchange:
				var data model.ChannelExec
				if err := json.Unmarshal(m.Body, &data); err != nil {
					wlog.Warn(fmt.Sprintf("Failed parse event content type: %v\n%s", m.ContentType, string(m.Body)))
				} else {
					a.execEvent <- data
				}
			default:
				wlog.Warn(fmt.Sprintf("unable to parse event, not found exchange %s", m.Exchange))
			}
			m.Ack(false)
		}

		if !a.stopping {
			a.initConnection()
		}
	}()
}

func (a *AMQP) handleCallMessage(data []byte) {
	callAction := model.CallActionData{}
	if err := json.Unmarshal(data, &callAction); err != nil {
		wlog.Error(fmt.Sprintf("parse error: %s", err.Error()))
		return
	}
	wlog.Debug(fmt.Sprintf("call %s [%s] ", callAction.Id, callAction.Event))
	a.callEvent <- callAction
}

func (a *AMQP) Close() {
	wlog.Debug("AMQP receive stop client")
	a.stopping = true
	if a.channel != nil {
		a.channel.Close()
		wlog.Debug("Close AMQP channel")
	}

	if a.connection != nil {
		a.connection.Close()
		wlog.Debug("Close AMQP connection")
	}
}

func (a *AMQP) getChannel() *amqp.Channel {
	a.RLock()
	defer a.RUnlock()

	return a.channel
}

func (a *AMQP) SendJSON(exchange string, key string, data []byte) *model.AppError {
	channel := a.getChannel()
	if channel == nil {
		return model.NewAppError("MQ", "mq.publish.channel.err", nil, "Not found publish channel", http.StatusInternalServerError)
	}
	err := channel.Publish(exchange, key, false, false, amqp.Publishing{
		ContentType: "text/json",
		Body:        data,
	})

	if err != nil {
		return model.NewAppError("MQ", "mq.publish.err", nil, err.Error(), http.StatusInternalServerError)
	}

	return nil
}

func (a *AMQP) ConsumeCallEvent() <-chan model.CallActionData {
	return a.callEvent
}

func (a *AMQP) ConsumeExec() <-chan model.ChannelExec {
	return a.execEvent
}
