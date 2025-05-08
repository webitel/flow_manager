package app

import (
	"fmt"
	"sync"
	"time"

	"github.com/webitel/wlog"

	"github.com/webitel/engine/pkg/discovery"
	"github.com/webitel/flow_manager/model"
)

const (
	pollingRemoveExpiredNumbers = 5 * 1000 // 30 sec
)

type listWatcher struct {
	fm        *FlowManager
	startOnce sync.Once
	watcher   *discovery.Watcher
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
			c.watcher = discovery.MakeWatcher("list-communications", pollingRemoveExpiredNumbers, c.cleanExpiredNumbers)
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

func (fm *FlowManager) ListCheckNumber(domainId int64, number string, listId *int, listName *string) (bool, *model.AppError) {
	return fm.Store.List().CheckNumber(domainId, number, listId, listName)
}

func (fm *FlowManager) ListAddCommunication(domainId int64, search *model.SearchEntity, comm *model.ListCommunication) *model.AppError {
	return fm.Store.List().AddDestination(domainId, search, comm)
}
