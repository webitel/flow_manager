package flow

import (
	"context"
	"fmt"
	"github.com/webitel/wlog"
	"time"
)

type TimerArgs struct {
	Name     string
	Interval int           `json:"interval"`
	Tries    int           `json:"tries"`
	Offset   int           `json:"offset"`
	Actions  []interface{} `json:"actions"`
}

func (scope *Flow) Timer(ctx context.Context, t TimerArgs, r Handler) {
	if t.Interval == 0 {
		// TODO set default ?
		return
	}

	if t.Tries == 0 {
		// todo set default ?
		t.Tries = 999
	}

	interval := time.Duration(t.Interval)
	timer := time.NewTimer(time.Second * interval)
	tries := 0
	defer wlog.Debug(fmt.Sprintf("timer [%s] stopped", t.Name))
	f := scope.Fork(t.Name, ArrInterfaceToArrayApplication(t.Actions))

	for {
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
			tries++
			Route(ctx, f, r)

			interval += time.Duration(t.Offset)
			if tries >= t.Tries || interval < 1 {
				timer.Stop()
				return
			}
			timer = time.NewTimer(time.Second * interval)
		}
	}
}
