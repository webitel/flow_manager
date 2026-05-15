package event

import (
	"context"
	"fmt"
	"net/http"
	"strconv"

	bscfg "github.com/webitel/flow_manager/internal/bootstrap/config"
	"github.com/webitel/flow_manager/internal/domain/call"
	"github.com/webitel/flow_manager/internal/domain/notification"
	"github.com/webitel/flow_manager/internal/domain/shared/ports"
	apperrs "github.com/webitel/flow_manager/internal/infrastructure/errors"
	"github.com/webitel/flow_manager/internal/infrastructure/utils"
	"github.com/webitel/flow_manager/internal/storage"
	"github.com/webitel/wlog"
)

const (
	engineExchange            = "engine"
	actionOpenLink            = "open_link"
	descTrackAppName          = "desc_track"
	refreshMissedNotification = "refresh_missed"
)

var ErrAllowUseMQ = apperrs.New(http.StatusForbidden, "App: app.settings.mq.allow_use.disabled: Allow push message to MQ is disabled")

// Adapter implements event-bus–backed Deps methods: notifications, open-link,
// MQ publishing.
type EventBusAdapter struct {
	bus    ports.EventBus
	store  storage.Store
	config *bscfg.Config
}

func NewEventBusAdapter(bus ports.EventBus, st storage.Store, cfg *bscfg.Config) *EventBusAdapter {
	return &EventBusAdapter{bus: bus, store: st, config: cfg}
}

func (a *EventBusAdapter) UserNotification(n notification.Notification) {
	if err := a.bus.Publish(context.Background(), engineExchange, "notification."+strconv.Itoa(int(n.DomainId)), n.ToJson()); err != nil {
		wlog.Error(err.Error())
	}
}

func (a *EventBusAdapter) NotificationMissedCalls(c call.MissedCall) {
	a.UserNotification(notification.Notification{
		DomainId:  c.DomainId,
		Action:    refreshMissedNotification,
		CreatedAt: utils.GetMillis(),
		ForUsers:  []int64{c.UserId},
		Body:      map[string]interface{}{"call_id": c.Id},
	})
}

func (a *EventBusAdapter) PushOpenLink(domainId int64, sockId string, userId int64, message, url string) error {
	return a.OpenLink(domainId, sockId, userId, message, url)
}

func (a *EventBusAdapter) OpenLink(domainId int64, sockId string, userId int64, message string, url string) error {
	if sockId == "" {
		sockSession, storeErr := a.store.SocketSession().Get(userId, domainId, descTrackAppName)
		if storeErr != nil {
			return fmt.Errorf("open_link: store.open_link.error: %w", storeErr)
		}
		sockId = sockSession.ID
	}
	n := notification.Notification{
		DomainId:  domainId,
		Action:    actionOpenLink,
		CreatedAt: utils.GetMillis(),
		ForUsers:  []int64{userId},
		SockID:    sockId,
		Body:      map[string]interface{}{"url": url, "message": message},
	}
	if pubErr := a.bus.Publish(context.Background(), engineExchange, "notification."+strconv.Itoa(int(n.DomainId)), n.ToJson()); pubErr != nil {
		wlog.Error(pubErr.Error())
		return fmt.Errorf("open_link: mq.publish.err: %w", pubErr)
	}
	return nil
}

// ConsumeCallEvent satisfies call_watcher.CallEventDeps.
func (a *EventBusAdapter) ConsumeCallEvent() <-chan call.CallActionData {
	return a.bus.ConsumeCallEvent()
}

func (a *EventBusAdapter) SendMQJson(exchange, key string, body []byte) error {
	if !a.config.AllowUseMQ {
		return ErrAllowUseMQ
	}
	if err := a.bus.Publish(context.Background(), exchange, key, body); err != nil {
		return fmt.Errorf("MQ: mq.publish.err: %w", err)
	}
	return nil
}
