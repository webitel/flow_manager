package grpc

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/webitel/flow_manager/gen/workflow"
	"github.com/webitel/flow_manager/infra/discovery"
	"github.com/webitel/flow_manager/model"
	"github.com/webitel/wlog"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Config struct {
	Host     string
	Port     int
	NodeName string
}

type Server struct {
	cfg             *Config
	server          *grpc.Server
	didFinishListen chan struct{}
	consume         chan model.Connection
	chatApi         *chatApi
	processingApi   *processingApi
	startOnce       sync.Once
	chatManager     *ChatManager
	nodeName        string
	workflow.UnsafeFlowServiceServer
	cb CallbackResolver

	log *wlog.Logger
}

type CallbackResolver interface {
	Callback(ctx context.Context, id string, data any) (any, error)
}

func NewServer(cfg *Config, cm *ChatManager, cb CallbackResolver) *Server {
	srv := &Server{
		cfg:             cfg,
		didFinishListen: make(chan struct{}),
		consume:         make(chan model.Connection),
		nodeName:        cfg.NodeName,
		chatManager:     cm,
		log: wlog.GlobalLogger().With(
			wlog.Namespace("context"),
			wlog.String("scope", "grpc server"),
		),
		cb: cb,
	}
	srv.chatApi = NewChatApi(srv)
	srv.processingApi = NewProcessingApi(srv)

	return srv
}

func publicAddr(lis net.Listener) (string, int) {
	h, p, _ := net.SplitHostPort(lis.Addr().String())
	if h == "::" {
		h = model.GetPublicAddr()
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

	workflow.RegisterFlowServiceServer(s.server, s)
	workflow.RegisterFlowChatServerServiceServer(s.server, s.chatApi)
	workflow.RegisterFlowProcessingServiceServer(s.server, s.processingApi)

	s.cfg.Host, s.cfg.Port = publicAddr(lis)

	go s.listen(lis)

	return nil
}

func (s *Server) listen(lis net.Listener) {
	defer s.log.Debug(fmt.Sprintf("close server listening"))
	s.log.Debug(fmt.Sprintf("server listening %s", lis.Addr().String()))
	err := s.server.Serve(lis)
	if err != nil {
		//FIXME
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

func (s Server) Name() string {
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
func (s *Server) Consume() <-chan model.Connection {
	return s.consume
}

func (s *Server) NodeName() string {
	return s.nodeName
}

func (s Server) Type() model.ConnectionType {
	return model.ConnectionTypeGrpc
}

func unaryInterceptor(ctx context.Context,
	req interface{},
	info *grpc.UnaryServerInfo,
	handler grpc.UnaryHandler) (interface{}, error) {
	start := time.Now()

	h, err := handler(ctx, req)

	log := wlog.GlobalLogger().With(wlog.Namespace("context"),
		wlog.Duration("duration", time.Since(start)),
		wlog.String("method", info.FullMethod),
	)

	if err != nil {
		log.Err(err)

		switch err.(type) {
		case *model.AppError:
			e := err.(*model.AppError)
			return h, status.Error(httpCodeToGrpc(e.StatusCode), e.ToJson())
		default:
			return h, err
		}
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
