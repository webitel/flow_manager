// Package list_watcher periodically removes expired list communications from
// the database.
package list_watcher

import (
	"fmt"
	"sync"
	"time"

	"github.com/webitel/wlog"

	"github.com/webitel/flow_manager/infra/watcher"
	"github.com/webitel/flow_manager/store"
)

const (
	pollingRemoveExpiredNumbers = 5 * 1000 // 5 sec in ms
)

// Worker polls the store and removes expired list-communication numbers.
type Worker struct {
	store     store.Store
	startOnce sync.Once
	watcher   *watcher.Watcher
	log       *wlog.Logger
}

// New creates a Worker backed by st.
func New(st store.Store, log *wlog.Logger) *Worker {
	return &Worker{
		store: st,
		log: log.With(
			wlog.Namespace("context"),
			wlog.String("scope", "list watcher"),
		),
	}
}

func (c *Worker) Start() {
	c.startOnce.Do(func() {
		go func() {
			c.watcher = watcher.MakeWatcher("list-communications", pollingRemoveExpiredNumbers, c.cleanExpiredNumbers)
			c.watcher.Start()
		}()
	})
}

func (c *Worker) Stop() {
	if c.watcher != nil {
		c.watcher.Stop()
	}
}

func (c *Worker) cleanExpiredNumbers() {
	count, err := c.store.List().CleanExpired()
	if err != nil {
		c.log.Error(err.Error())
		time.Sleep(time.Second * 5)
	}

	if count > 0 {
		c.log.Debug(fmt.Sprintf("removed %d expired numbers", count))
	}
}
