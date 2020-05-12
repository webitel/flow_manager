package web_chat

import (
	"fmt"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/webitel/flow_manager/model"
	"github.com/webitel/wlog"
	"net"
	"net/http"
	"sync"
	"time"
)

type server struct {
	app             App
	host            string
	port            int
	didFinishListen chan struct{}
	consume         chan model.Connection
	startOnce       sync.Once

	Server     *http.Server
	ListenAddr *net.TCPAddr
	RootRouter *mux.Router
	Router     *mux.Router
}

func NewServer(app App, host string, port int) model.Server {
	s := &server{
		app:        app,
		host:       host,
		port:       port,
		consume:    make(chan model.Connection),
		RootRouter: mux.NewRouter(),
	}
	s.InitApi()
	return s
}

func (s *server) Name() string {
	return "WebChat"
}

func (s *server) Start() *model.AppError {
	var handler http.Handler = &CorsWrapper{s.RootRouter}
	s.Server = &http.Server{
		Handler: handlers.RecoveryHandler(handlers.RecoveryLogger(&RecoveryLogger{}), handlers.PrintRecoveryStack(true))(handler),
	}

	addr := fmt.Sprintf("%s:%d", s.host, s.port)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return model.NewAppError("Chat", "chat.server.start", nil, err.Error(), http.StatusInternalServerError)
	}

	s.ListenAddr = listener.Addr().(*net.TCPAddr)
	s.didFinishListen = make(chan struct{})

	go func() {
		var err error
		defer wlog.Debug(fmt.Sprintf("[WebChat] close server listening"))
		wlog.Debug(fmt.Sprintf("[WebChat] server listening %s", s.ListenAddr.String()))
		err = s.Server.Serve(listener)
		if err != nil && err != http.ErrServerClosed {
			wlog.Critical(fmt.Sprintf("error starting server, err:%v", err))
			time.Sleep(time.Second)
		}
		close(s.didFinishListen)
	}()

	return nil
}

func (s *server) Stop() {
	s.Server.Close()
	<-s.didFinishListen
}

func (s *server) Host() string {
	return s.host
}

func (s *server) Port() int {
	return s.port
}

func (s *server) Consume() <-chan model.Connection {
	return s.consume
}

func (s *server) Type() model.ConnectionType {
	return model.ConnectionTypeWebChat
}
