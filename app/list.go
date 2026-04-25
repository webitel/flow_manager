package app

import (
	"fmt"
	"net/http"
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

func (fm *FlowManager) ListCheckNumber(domainId int64, number string, listId *int, listName *string) (bool, *model.AppError) {
	ok, err := fm.Store.List().CheckNumber(domainId, number, listId, listName)
	if err != nil {
		return false, model.NewAppError("ListCheckNumber", "store.list.check_number", nil, err.Error(), http.StatusInternalServerError)
	}
	return ok, nil
}

func (fm *FlowManager) ListAddCommunication(domainId int64, search *model.SearchEntity, comm *model.ListCommunication) *model.AppError {
	if err := fm.Store.List().AddDestination(domainId, search, comm); err != nil {
		return model.NewAppError("ListAddCommunication", "store.list.add_destination", nil, err.Error(), http.StatusInternalServerError)
	}
	return nil
}
