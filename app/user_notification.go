package app

import (
	"context"
	"fmt"
	"sync"

	"github.com/webitel/engine/pkg/discovery"
	"github.com/webitel/wlog"

	"github.com/webitel/flow_manager/model"
)

const (
	pollingRemoveExpiredUserNotifications = 60 * 1000
)

type userNotificationWatcher struct {
	fm        *FlowManager
	startOnce sync.Once
	watcher   *discovery.Watcher
	log       *wlog.Logger
}

func NewUserNotificationWatcher(fm *FlowManager) *userNotificationWatcher {
	return &userNotificationWatcher{
		fm: fm,
		log: fm.Log().With(
			wlog.Namespace("context"),
			wlog.String("scope", "user notification watcher"),
		),
	}
}

func (c *userNotificationWatcher) Start() {
	c.startOnce.Do(func() {
		go func() {
			c.watcher = discovery.MakeWatcher("user-notifications", pollingRemoveExpiredUserNotifications, c.cleanExpired)
			c.watcher.Start()
		}()
	})
}

func (c *userNotificationWatcher) Stop() {
	if c.watcher != nil {
		c.watcher.Stop()
	}
}

func (c *userNotificationWatcher) cleanExpired() {
	count, err := c.fm.Store.UserNotification().CleanExpired()
	if err != nil {
		c.log.Error(err.Error())

		return
	}

	if count > 0 {
		c.log.Debug(fmt.Sprintf("removed %d expired user notifications", count))
	}
}

func (fm *FlowManager) SaveUserNotification(ctx context.Context, n *model.UserNotification) *model.AppError {
	return fm.Store.UserNotification().Create(ctx, n)
}
