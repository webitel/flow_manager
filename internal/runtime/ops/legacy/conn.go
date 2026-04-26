package legacy

import (
	"context"
	"fmt"
	"sync"

	"github.com/webitel/flow_manager/model"
)

// varTrackingConn wraps model.Connection and records every variable written via Set.
type varTrackingConn struct {
	model.Connection
	mu      sync.Mutex
	written map[string]string
}

func newVarTrackingConn(conn model.Connection) *varTrackingConn {
	return &varTrackingConn{Connection: conn, written: make(map[string]string)}
}

func (c *varTrackingConn) Set(ctx context.Context, vars model.Variables) (model.Response, *model.AppError) {
	resp, err := c.Connection.Set(ctx, vars)
	if err == nil {
		c.mu.Lock()
		for k, v := range vars {
			c.written[k] = fmt.Sprintf("%v", v)
		}
		c.mu.Unlock()
	}
	return resp, err
}

// delta returns a snapshot of all variables written since creation.
func (c *varTrackingConn) delta() map[string]string {
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(c.written) == 0 {
		return nil
	}
	out := make(map[string]string, len(c.written))
	for k, v := range c.written {
		out[k] = v
	}
	return out
}
