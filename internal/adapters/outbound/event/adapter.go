package event

import (
	"context"
	"net/http"
	"strconv"

	"github.com/webitel/flow_manager/internal/domain/shared/ports"
	"github.com/webitel/flow_manager/model"
	"github.com/webitel/flow_manager/store"
	"github.com/webitel/wlog"
)

const (
	engineExchange            = "engine"
	actionOpenLink            = "open_link"
	descTrackAppName          = "desc_track"
	refreshMissedNotification = "refresh_missed"
)

var ErrAllowUseMQ = model.NewAppError("App", "app.settings.mq.allow_use.disabled", nil, "Allow push message to MQ is disabled", http.StatusForbidden)

// Adapter implements event-bus–backed Deps methods: notifications, open-link,
// MQ publishing.
type EventBusAdapter struct {
	bus    ports.EventBus
	store  store.Store
	config *model.Config
}

func NewEventBusAdapter(bus ports.EventBus, st store.Store, cfg *model.Config) *EventBusAdapter {
	return &EventBusAdapter{bus: bus, store: st, config: cfg}
}

func (a *EventBusAdapter) UserNotification(n model.Notification) {
	if err := a.bus.Publish(context.Background(), engineExchange, "notification."+strconv.Itoa(int(n.DomainId)), n.ToJson()); err != nil {
		wlog.Error(err.Error())
	}
}

func (a *EventBusAdapter) NotificationMissedCalls(call model.MissedCall) {
	a.UserNotification(model.Notification{
		DomainId:  call.DomainId,
		Action:    refreshMissedNotification,
		CreatedAt: model.GetMillis(),
		ForUsers:  []int64{call.UserId},
		Body:      map[string]interface{}{"call_id": call.Id},
	})
}

func (a *EventBusAdapter) PushOpenLink(domainId int64, sockId string, userId int64, message, url string) error {
	return a.OpenLink(domainId, sockId, userId, message, url)
}

func (a *EventBusAdapter) OpenLink(domainId int64, sockId string, userId int64, message string, url string) error {
	if sockId == "" {
		sockSession, storeErr := a.store.SocketSession().Get(userId, domainId, descTrackAppName)
		if storeErr != nil {
			return model.NewAppError("open_link", "store.open_link.error", nil, storeErr.Error(), http.StatusInternalServerError)
		}
		sockId = sockSession.ID
	}
	n := model.Notification{
		DomainId:  domainId,
		Action:    actionOpenLink,
		CreatedAt: model.GetMillis(),
		ForUsers:  []int64{userId},
		SockID:    sockId,
		Body:      map[string]interface{}{"url": url, "message": message},
	}
	if pubErr := a.bus.Publish(context.Background(), engineExchange, "notification."+strconv.Itoa(int(n.DomainId)), n.ToJson()); pubErr != nil {
		wlog.Error(pubErr.Error())
		return model.NewAppError("open_link", "mq.publish.err", nil, pubErr.Error(), http.StatusInternalServerError)
	}
	return nil
}

func (a *EventBusAdapter) SendMQJson(exchange, key string, body []byte) error {
	if !a.config.AllowUseMQ {
		return ErrAllowUseMQ
	}
	if err := a.bus.Publish(context.Background(), exchange, key, body); err != nil {
		return model.NewAppError("MQ", "mq.publish.err", nil, err.Error(), http.StatusInternalServerError)
	}
	return nil
}
