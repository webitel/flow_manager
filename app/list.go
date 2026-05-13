package app

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/webitel/wlog"

	"github.com/webitel/flow_manager/infra/watcher"
	"github.com/webitel/flow_manager/model"
)

const (
	pollingRemoveExpiredNumbers = 5 * 1000 // 30 sec
)

type listWatcher struct {
	fm        *FlowManager
	startOnce sync.Once
	watcher   *watcher.Watcher
	log       *wlog.Logger
}

func NewListWatcher(fm *FlowManager) *listWatcher {
	return &listWatcher{
		fm: fm,
		log: fm.Log().With(
			wlog.Namespace("context"),
			wlog.String("scope", "list watcher"),
		),
	}
}

func (c *listWatcher) Start() {
	c.startOnce.Do(func() {
		go func() {
			c.watcher = watcher.MakeWatcher("list-communications", pollingRemoveExpiredNumbers, c.cleanExpiredNumbers)
			c.watcher.Start()
		}()
	})
}

func (c *listWatcher) Stop() {
	if c.watcher != nil {
		c.watcher.Stop()
	}
}

func (c *listWatcher) cleanExpiredNumbers() {
	count, err := c.fm.Store.List().CleanExpired()
	if err != nil {
		c.log.Error(err.Error())
		time.Sleep(time.Second * 5)
	}

	if count > 0 {
		c.log.Debug(fmt.Sprintf("removed %d expired numbers", count))
	}
}

func (fm *FlowManager) CheckList(domainId int64, number string, listId *int, listName *string) (bool, error) {
	ok, appErr := fm.ListCheckNumber(domainId, number, listId, listName)
	if appErr != nil {
		return false, appErr
	}
	return ok, nil
}

func (fm *FlowManager) AddToList(ctx context.Context, domainId int64, listId *int, listName *string, destination string, description *string, expireAtMS int64) error {
	comm := &model.ListCommunication{
		Destination: destination,
		Description: description,
	}
	if expireAtMS > 0 {
		t := time.UnixMilli(expireAtMS)
		comm.ExpireAt = &t
	}
	appErr := fm.ListAddCommunication(domainId, &model.SearchEntity{Id: listId, Name: listName}, comm)
	if appErr != nil {
		return appErr
	}
	return nil
}
