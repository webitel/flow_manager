package grpc

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/webitel/engine/discovery"
	"github.com/webitel/engine/utils"
	"github.com/webitel/flow_manager/model"
	"github.com/webitel/protos/workflow"
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

type server struct {
	cfg             *Config
	server          *grpc.Server
	didFinishListen chan struct{}
	consume         chan model.Connection
	chatApi         *chatApi
	processingApi   *processingApi
	startOnce       sync.Once
	chatManager     *chatManager
	nodeName        string
	workflow.UnsafeFlowServiceServer
}

func NewServer(cfg *Config) model.Server {
	srv := &server{
		cfg:             cfg,
		didFinishListen: make(chan struct{}),
		consume:         make(chan model.Connection),
		nodeName:        cfg.NodeName,
	}
	srv.chatApi = NewChatApi(srv)
	srv.processingApi = NewProcessingApi(srv)

	return srv
}

func publicAddr(lis net.Listener) (string, int) {
	h, p, _ := net.SplitHostPort(lis.Addr().String())
	if h == "::" {
		h = utils.GetPublicAddr()
	}
	port, _ := strconv.Atoi(p)
	return h, port
}

//todo del me
func (s *server) Cluster(discovery discovery.ServiceDiscovery) *model.AppError {
	s.chatManager = NewChatManager(discovery)
	if err := s.chatManager.Start(); err != nil {
		return model.NewAppError("GRPC", "grpc.chat.client_manager.app_err", nil, err.Error(), http.StatusInternalServerError)
	}

	return nil
}

func (s *server) Start() *model.AppError {
	address := s.getAddress()
	lis, err := net.Listen("tcp", address)
	if err != nil {
		return model.NewAppError("GRPC", "grpc.start_server.error", nil, err.Error(), http.StatusInternalServerError)
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

func (s *server) listen(lis net.Listener) {
	defer wlog.Debug(fmt.Sprintf("[grpc] close server listening"))
	wlog.Debug(fmt.Sprintf("[grpc] server listening %s", lis.Addr().String()))
	err := s.server.Serve(lis)
	if err != nil {
		//FIXME
		panic(err.Error())
	} else {
		close(s.didFinishListen)
	}
}

func (s *server) getAddress() string {
	p := s.Port()
	h := s.Host()
	if p == 0 {
		return fmt.Sprintf("%s:", h)
	}
	return fmt.Sprintf("%s:%d", h, p)
}

func (s server) Name() string {
	return "GRPC"
}

func (s *server) Stop() {
	close(s.consume)
	s.server.Stop()
	<-s.didFinishListen
}

func (s *server) Host() string {
	return s.cfg.Host
}
func (s *server) Port() int {
	return s.cfg.Port
}
func (s *server) Consume() <-chan model.Connection {
	return s.consume
}

func (s *server) NodeName() string {
	return s.nodeName
}

func (s server) Type() model.ConnectionType {
	return model.ConnectionTypeGrpc
}

func unaryInterceptor(ctx context.Context,
	req interface{},
	info *grpc.UnaryServerInfo,
	handler grpc.UnaryHandler) (interface{}, error) {
	start := time.Now()

	h, err := handler(ctx, req)

	if err != nil {
		wlog.Error(fmt.Sprintf("method %s duration %s, error: %v", info.FullMethod, time.Since(start), err.Error()))

		switch err.(type) {
		case *model.AppError:
			e := err.(*model.AppError)
			return h, status.Error(httpCodeToGrpc(e.StatusCode), e.ToJson())
		default:
			return h, err
		}
	} else {
		wlog.Debug(fmt.Sprintf("method %s duration %s", info.FullMethod, time.Since(start)))
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
