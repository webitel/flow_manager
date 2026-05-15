// Package servers groups all protocol-level servers and manages their
// start/stop lifecycle.
package servers

import (
	"fmt"

	"github.com/webitel/wlog"

	fmgrpc "github.com/webitel/flow_manager/internal/adapters/inbound/grpc"
	"github.com/webitel/flow_manager/internal/domain/flow"
)

// grpcConsumer is a transport that exposes only a consume channel (no lifecycle).
type grpcConsumer interface {
	Consume() <-chan flow.Connection
}

// Servers groups all protocol-level servers so they can be injected as a
// single value, avoiding the fx same-type ambiguity for multiple flow.Server
// providers.
type Servers struct {
	Grpc     *fmgrpc.Server
	CallGrpc grpcConsumer
	Esl      flow.Server
	Mail     flow.Server
	Channel  flow.Server
	Im       flow.Server
	Http     flow.Server // nil when WebHook.Addr is not configured
}

// Register starts all non-nil servers in order.
func (s Servers) Register() error {
	servers := []flow.Server{s.Grpc, s.Esl, s.Mail, s.Channel, s.Im}

	for _, v := range servers {
		if err := startServer(v); err != nil {
			return err
		}
	}

	return nil
}

// Stop stops all non-nil servers in order.
func (s Servers) Stop() {
	servers := []flow.Server{s.Grpc, s.Esl, s.Mail, s.Channel, s.Im}

	for _, v := range servers {
		stopServer(v)
	}
}

func startServer(s flow.Server) error {
	if s == nil {
		return nil
	}
	if err := s.Start(); err != nil {
		return err
	}

	wlog.Info(fmt.Sprintf("started [%s] server", s.Name()))
	return nil
}

func stopServer(s flow.Server) {
	if s == nil {
		return
	}
	s.Stop()

	wlog.Info(fmt.Sprintf("stopped [%s] server", s.Name()))
}
