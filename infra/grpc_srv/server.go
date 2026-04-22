package grpc_srv

import (
	"context"
	"fmt"
	"net"
	"os"
	"strconv"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/webitel/wlog"
)

const RequestContextName = "grpc_ctx"

type RequestContextSessionKey struct{}

var ErrUnauthenticated = status.Error(codes.Unauthenticated, "Unauthenticated")

type Server struct {
	*grpc.Server

	Addr     string
	host     string
	port     int
	log      *wlog.Logger
	listener net.Listener
}

// New provides a new gRPC server.
func New(addr string, log *wlog.Logger) (*Server, error) {
	s := grpc.NewServer(
		grpc.ChainUnaryInterceptor(),
		grpc.UnaryInterceptor(unaryInterceptor(log)),
	)

	l, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}

	h, p, err := net.SplitHostPort(l.Addr().String())
	if err != nil {
		return nil, err
	}

	port, _ := strconv.Atoi(p)

	if h == "::" {
		h = publicAddr()
	}

	return &Server{
		Addr:     addr,
		Server:   s,
		log:      log,
		host:     h,
		port:     port,
		listener: l,
	}, nil
}

func (s *Server) Listen() error {
	return s.Serve(s.listener)
}

func (s *Server) Shutdown() error {
	s.log.Debug("receive shutdown grpc")
	err := s.listener.Close()
	s.GracefulStop()

	return err
}

func (s *Server) Host() string {
	if e, ok := os.LookupEnv("PROXY_GRPC_HOST"); ok {
		return e
	}

	return s.host
}

func (s *Server) Port() int {
	return s.port
}

func publicAddr() string {
	ifaces, err := net.Interfaces()
	if err != nil {
		return ""
	}

	for _, i := range ifaces {
		addrs, err := i.Addrs()
		if err != nil {
			continue
		}

		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			default:
				continue
			}

			if isPublicIP(ip) {
				return ip.String()
			}
			// process IP address
		}
	}

	return ""
}

func isPublicIP(ip net.IP) bool {
	if ip.IsLoopback() || ip.IsLinkLocalMulticast() || ip.IsLinkLocalUnicast() {
		return false
	}

	return true
}

func unaryInterceptor(log *wlog.Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context,
		req any,
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (any, error) {
		start := time.Now()

		h, err := handler(ctx, req)

		l := log.With(wlog.String("method", info.FullMethod))

		if err != nil {
			l.Error(err.Error(), wlog.Float64("duration_ms", float64(time.Since(start).Microseconds())/float64(1000)))
		} else {
			l.Debug(fmt.Sprintf("[OK] %s", info.FullMethod), wlog.Float64("duration_ms", float64(time.Since(start).Microseconds())/float64(1000)))
		}

		return h, err
	}
}
