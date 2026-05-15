package grpc

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/webitel/wlog"

	workflow2 "github.com/webitel/flow_manager/api/gen/workflow"
	"github.com/webitel/flow_manager/internal/domain/flow"
	"github.com/webitel/flow_manager/internal/infrastructure/discovery"
	apperrs "github.com/webitel/flow_manager/internal/infrastructure/errors"
	"github.com/webitel/flow_manager/internal/infrastructure/utils"
)

type Config struct {
	Host     string
	Port     int
	NodeName string
}

type serviceEntry struct {
	desc *grpc.ServiceDesc
	impl any
}

type Server struct {
	cfg             *Config
	server          *grpc.Server
	didFinishListen chan struct{}
	consume         chan flow.Connection
	services        []serviceEntry
	startOnce       sync.Once
	nodeName        string

	log *wlog.Logger
}

func NewServer(cfg *Config, cm *ChatManager) *Server {
	srv := &Server{
		cfg:             cfg,
		didFinishListen: make(chan struct{}),
		consume:         make(chan flow.Connection),
		nodeName:        cfg.NodeName,
		log: wlog.GlobalLogger().With(
			wlog.Namespace("context"),
			wlog.String("scope", "grpc server"),
		),
	}
	srv.Register(&workflow2.FlowChatServerService_ServiceDesc, newChatApi(srv.Sink(), cm))
	srv.Register(&workflow2.FlowProcessingService_ServiceDesc, newProcessingApi(srv.Sink(), cfg.NodeName))
	return srv
}

// Register adds a gRPC service to be served. Must be called before Start.
func (s *Server) Register(desc *grpc.ServiceDesc, impl any) {
	s.services = append(s.services, serviceEntry{desc, impl})
}

// Sink returns a write-only channel that gRPC service handlers use to enqueue
// inbound connections for the Dispatcher.
func (s *Server) Sink() chan<- flow.Connection {
	return s.consume
}

func publicAddr(lis net.Listener) (string, int) {
	h, p, _ := net.SplitHostPort(lis.Addr().String())
	if h == "::" {
		h = utils.GetPublicAddr()
	}
	port, _ := strconv.Atoi(p)
	return h, port
}

// todo del me
func (s *Server) Cluster(discovery discovery.ServiceDiscovery) error {
	return nil
}

func (s *Server) Start() error {
	address := s.getAddress()
	lis, err := net.Listen("tcp", address)
	if err != nil {
		return fmt.Errorf("grpc.start_server.error: %w", err)
	}

	s.server = grpc.NewServer(
		grpc.UnaryInterceptor(unaryInterceptor),
	)

	for _, svc := range s.services {
		s.server.RegisterService(svc.desc, svc.impl)
	}

	s.cfg.Host, s.cfg.Port = publicAddr(lis)

	go s.listen(lis)

	return nil
}

func (s *Server) listen(lis net.Listener) {
	defer s.log.Debug(fmt.Sprintf("close server listening"))
	s.log.Debug(fmt.Sprintf("server listening %s", lis.Addr().String()))
	err := s.server.Serve(lis)
	if err != nil {
		// FIXME
		panic(err.Error())
	} else {
		close(s.didFinishListen)
	}
}

func (s *Server) getAddress() string {
	p := s.Port()
	h := s.Host()
	if p == 0 {
		return fmt.Sprintf("%s:", h)
	}
	return fmt.Sprintf("%s:%d", h, p)
}

func (s *Server) Name() string {
	return "GRPC"
}

func (s *Server) Stop() {
	close(s.consume)
	s.server.Stop()
	<-s.didFinishListen
}

func (s *Server) Host() string {
	return s.cfg.Host
}

func (s *Server) Port() int {
	return s.cfg.Port
}

func (s *Server) Consume() <-chan flow.Connection {
	return s.consume
}

func (s *Server) NodeName() string {
	return s.nodeName
}

func (s *Server) Type() flow.ConnectionType {
	return flow.ConnectionTypeGrpc
}

func unaryInterceptor(ctx context.Context,
	req any,
	info *grpc.UnaryServerInfo,
	handler grpc.UnaryHandler,
) (any, error) {
	start := time.Now()

	h, err := handler(ctx, req)

	log := wlog.GlobalLogger().With(wlog.Namespace("context"),
		wlog.Duration("duration", time.Since(start)),
		wlog.String("method", info.FullMethod),
	)

	if err != nil {
		log.Err(err)

		return h, status.Error(httpCodeToGrpc(apperrs.CodeOf(err)), err.Error())
	} else {
		log.Debug(info.FullMethod + " - OK")
	}

	return h, err
}

func httpCodeToGrpc(c int) codes.Code {
	switch c {
	case http.StatusBadRequest:
		return codes.InvalidArgument
	case http.StatusAccepted:
		return codes.ResourceExhausted
	case http.StatusUnauthorized:
		return codes.Unauthenticated
	case http.StatusForbidden:
		return codes.PermissionDenied
	default:
		return codes.Internal
	}
}
