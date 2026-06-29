package rabbit

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/webitel/flow_manager/model"
	"github.com/webitel/wlog"
)

func (a *AMQP) subscribeIM() {
	imQueueName := fmt.Sprintf("%s.%s.any", model.IMQueueNamePrefix, model.NewId()[0:8])
	queueArgs := amqp.Table{"x-queue-type": "quorum", "x-expires": 10000}

	imQueue, err := a.channel.QueueDeclare(imQueueName, true, false, false, true, queueArgs)
	if err != nil {
		wlog.Critical("[AQMP] declare IM queue", wlog.String("queue", imQueueName), wlog.Err(err))
		panic("failed to declare AMQP IM queue")
	}

	wlog.Debug("[AMQP] successfully declared IM AMQP queue", wlog.String("queue", imQueue.Name), wlog.Int("consumers", imQueue.Consumers))

	if err = a.channel.QueueBind(imQueue.Name, "#", model.IMExchange, true, nil); err != nil {
		wlog.Critical("[AMQP] during binding IM queue to delivery exchange", wlog.String("queue", imQueue.Name), wlog.String("exchange", model.IMExchange), wlog.Err(err))
		panic("error during binding IM queue to delivery exchange")
	}

	msgs, err := a.channel.Consume(imQueue.Name, "", false, false, false, false, nil)
	if err != nil {
		wlog.Critical("[AMQP] during starting consuming IM messages", wlog.String("queue", imQueue.Name), wlog.String("exchange", model.IMExchange), wlog.Err(err))
		panic("error during starting consuming IM messages")
	}

	for m := range msgs {
		if m.Exchange != model.IMExchange {
			wlog.Warn("[AMQP] received message from unexpected exchange", wlog.String("received_exchange", m.Exchange), wlog.String("expected_exchange", model.IMExchange))

			continue
		}

		if err := a.processReceivedIMEvent(m); err != nil {
			wlog.Error("[AMQP] processing received IM event", wlog.String("received_rk", m.RoutingKey), wlog.Err(err))
		}

		m.Ack(false)
	}

	if !a.stopping {
		a.initConnection()
	}
}

func (a *AMQP) processReceivedIMEvent(event amqp.Delivery) error {
	rk := event.RoutingKey

	if strings.HasPrefix(rk, "im_delivery.v1.") && strings.HasSuffix(rk, ".message.created") {
		return a.handleIMMessageEvent(event)
	}

	if strings.HasPrefix(rk, "im_delivery.v1.") && strings.HasSuffix(rk, ".interactive_callback.reacted") {
		return a.handleIMInteractiveCallbackEvent(event)
	}

	if strings.HasPrefix(rk, "im_delivery.v1.") && strings.HasSuffix(rk, ".bot.control.released") {
		return a.handleIMBotControlReleasedEvent(event)
	}

	return nil
}

const XJWTPayloadAMQPHeader string = "x-jwt-payload"

func tryExtractJWTPayloadFromHeader(headers amqp.Table) (string, error) {
	rawHeader, ok := headers[XJWTPayloadAMQPHeader]
	if !ok {
		return "", nil
	}

	var jwtHeaderStr string
	switch v := rawHeader.(type) {
	case string:
		jwtHeaderStr = v
	case []byte:
		jwtHeaderStr = string(v)
	default:
		return "", model.NewRequestError("tryExtractJWTPayloadFromHeader", fmt.Sprintf("invalid jwt header type: %T", v))
	}

	if jwtHeaderStr == "" {
		return "", model.NewRequestError("tryExtractJWTPayloadFromHeader", "empty jwt header value")
	}

	encodedHeader, err := base64.RawURLEncoding.DecodeString(jwtHeaderStr)
	if err != nil {
		return "", model.NewInternalError("tryExtractJWTPayloadFromHeader", fmt.Sprintf("decoding jwt header base64: %v", err))
	}

	return string(encodedHeader), nil
}

func (a *AMQP) handleIMMessageEvent(event amqp.Delivery) error {
	var messageWrapper model.MessageWrapper[model.Message]
	if err := json.Unmarshal(event.Body, &messageWrapper); err != nil {
		return model.NewAppError(
			"processReceivedIMEvent",
			"rabbit.client.process_received_im_event.unmarshaling_body",
			nil,
			err.Error(),
			http.StatusBadRequest,
		)
	}

	if messageWrapper.Echo {
		wlog.Info("skipping echo IM event", wlog.String("thread_id", messageWrapper.Message.ThreadID), wlog.String("message_id", messageWrapper.Message.ID))

		return nil
	}

	jwtPayload, err := tryExtractJWTPayloadFromHeader(event.Headers)
	if err != nil {
		wlog.Error("extracting jwt payload from event headers", wlog.Err(err))
	}

	if jwtPayload != "" {
		messageWrapper.SetJWTPayload(jwtPayload)
	}

	messageWrapper.Type = model.IMEventTypeMessage

	a.imEvents <- messageWrapper

	return nil
}

func (a *AMQP) handleIMInteractiveCallbackEvent(event amqp.Delivery) error {
	var interactiveCallbackWrapper model.MessageWrapper[model.InteractiveCallback]
	if err := json.Unmarshal(event.Body, &interactiveCallbackWrapper); err != nil {
		return model.NewAppError(
			"handleIMInteractiveCallbackEvent",
			"rabbit.client_im.handle_im_interactive_callback_event.unmarshal_event",
			nil,
			err.Error(),
			http.StatusBadRequest,
		)
	}

	jwtPayload, err := tryExtractJWTPayloadFromHeader(event.Headers)
	if err != nil {
		wlog.Error("extracting jwt payload from event headers", wlog.Err(err))
	}

	if jwtPayload != "" {
		interactiveCallbackWrapper.SetJWTPayload(jwtPayload)
	}

	interactiveCallbackWrapper.Type = model.IMEventTypeCallback

	a.imEvents <- interactiveCallbackWrapper

	return nil
}

func (a *AMQP) handleIMBotControlReleasedEvent(event amqp.Delivery) error {
	var wrapper model.MessageWrapper[model.BotControlReleased]
	if err := json.Unmarshal(event.Body, &wrapper); err != nil {
		return model.NewAppError(
			"handleIMBotControlReleasedEvent",
			"rabbit.client_im.handle_im_bot_control_released_event.unmarshal_event",
			nil,
			err.Error(),
			http.StatusBadRequest,
		)
	}

	wrapper.Type = model.IMEventTypeBotControlReleased

	a.imEvents <- wrapper

	return nil
}
