package grpc

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/netip"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/webitel/engine/pkg/discovery"
	"github.com/webitel/flow_manager/gen/workflow"
	"github.com/webitel/flow_manager/model"
	"github.com/webitel/webitel-go-kit/pkg/hostaddr"
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
	listener        net.Listener
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

func NewServer(cfg *Config, cm *ChatManager, cb CallbackResolver) *server {
	srv := &server{
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

// todo del me
func (s *server) Cluster(discovery discovery.ServiceDiscovery) *model.AppError {
	return nil
}

func (s *server) Start() *model.AppError {
	host := strings.Trim(s.Host(), "[]")
	if host == "" {
		ip, err := hostaddr.PrivateIPv4()
		switch {
		case errors.Is(err, hostaddr.ErrNoPrivateIPv4):
			host = "127.0.0.1"
			s.log.Warn("no private IPv4 address found, grpc listens on loopback; set GRPC_ADDR to accept remote connections", wlog.String("address", host))
		case err != nil:
			return model.NewAppError("GRPC", "grpc.start_server.error", nil, fmt.Sprintf("resolve default grpc address: %s; set GRPC_ADDR explicitly", err), http.StatusInternalServerError)
		default:
			host = ip.String()
		}
	}

	lis, err := net.Listen("tcp", net.JoinHostPort(host, strconv.Itoa(s.Port())))
	if err != nil {
		return model.NewAppError("GRPC", "grpc.start_server.error", nil, err.Error(), http.StatusInternalServerError)
	}

	s.server = grpc.NewServer(
		grpc.UnaryInterceptor(unaryInterceptor),
	)

	workflow.RegisterFlowServiceServer(s.server, s)
	workflow.RegisterFlowChatServerServiceServer(s.server, s.chatApi)
	workflow.RegisterFlowProcessingServiceServer(s.server, s.processingApi)

	listen, err := netip.ParseAddrPort(lis.Addr().String())
	if err != nil {
		lis.Close()
		return model.NewAppError("GRPC", "grpc.start_server.error", nil, err.Error(), http.StatusInternalServerError)
	}
	advertise, err := hostaddr.Advertise(listen)
	if err != nil {
		lis.Close()
		return model.NewAppError("GRPC", "grpc.start_server.error", nil, fmt.Sprintf("resolve grpc advertise address: %s; set GRPC_ADDR to a specific IP", err), http.StatusInternalServerError)
	}

	s.cfg.Host, s.cfg.Port = advertise.Addr().String(), int(advertise.Port())
	s.listener = lis

	go s.listen(lis)

	return nil
}

func (s *server) Listener() net.Listener {
	return s.listener
}

func (s *server) listen(lis net.Listener) {
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
