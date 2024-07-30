package chat_ai

import (
	"github.com/webitel/engine/utils"
	"golang.org/x/sync/singleflight"
	"google.golang.org/grpc"
	"time"
)

var requestGroup singleflight.Group

type Hub struct {
	connections utils.ObjectCache
}

func NewHub() *Hub {
	return &Hub{
		connections: utils.NewLru(100), // TODO add LRU hook delete
	}
}

func (h *Hub) GetClient(addr string) (*Client, error) {
	c, ok := h.connections.Get(addr)
	if ok {
		return c.(*Client), nil
	}

	res, err, shared := requestGroup.Do(addr, func() (interface{}, error) {
		return h.connectClient(addr)
	})

	if err != nil {
		return nil, err
	}

	if !shared {
		h.connections.Add(addr, res.(*Client))
	}

	return res.(*Client), nil
}

func (h *Hub) connectClient(addr string) (*Client, error) {
	conn, err := grpc.NewClient(addr, grpc.WithInsecure(), grpc.WithBlock(), grpc.WithTimeout(2*time.Second))
	if err != nil {
		return nil, err
	}

	cli := NewClient(conn)

	h.connections.Add(addr, cli)
	return cli, nil
}
