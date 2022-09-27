package email

import (
	"sync"
	"time"

	"github.com/webitel/engine/discovery"
	"github.com/webitel/engine/utils"
	"github.com/webitel/flow_manager/model"
	"github.com/webitel/flow_manager/store"
	"github.com/webitel/wlog"
)

var (
	FetchProfileInterval = time.Second * 20
	SizeCache            = 1000
)

type server struct {
	store           store.EmailStore
	profiles        *utils.Cache
	didFinishListen chan struct{}
	stopped         chan struct{}
	startOnce       sync.Once
	consume         chan model.Connection
}

func New(s store.EmailStore) model.Server {
	return &server{
		store:           s,
		profiles:        utils.NewLru(SizeCache),
		didFinishListen: make(chan struct{}),
		stopped:         make(chan struct{}),
		consume:         make(chan model.Connection),
	}
}

func (s *server) Name() string {
	return "Email"
}

func (s *server) Cluster(discovery discovery.ServiceDiscovery) *model.AppError {
	return nil
}

func (s *server) Start() *model.AppError {
	s.startOnce.Do(func() {
		go s.listen()
	})
	return nil
}

func (s *server) Stop() {
	close(s.didFinishListen)
	<-s.stopped
}

func (s *server) Host() string {
	return "" //TODO
}

func (s *server) Port() int {
	return 0
}

func (s *server) Type() model.ConnectionType {
	return model.ConnectionTypeEmail
}

func (s *server) Consume() <-chan model.Connection {
	return s.consume
}

func (s *server) listen() {
	defer func() {
		wlog.Debug("stop listen email server...") //TODO
		close(s.stopped)
	}()
	wlog.Debug("start listen emails")
	for {
		select {
		case <-s.didFinishListen:
			return
		case <-time.After(FetchProfileInterval):
			tasks, err := s.store.ProfileTaskFetch("")
			if err != nil {
				wlog.Error(err.Error())
				time.Sleep(time.Second * 5)
			} else {
				for _, v := range tasks {
					s.fetchNewMessageInProfile(v)
				}
			}
		}
	}
}

func (s *server) GetProfile(p *model.EmailProfileTask) (*Profile, *model.AppError) {
	var pp *Profile
	profile, ok := s.profiles.Get(p.Id)
	if ok {
		pp = profile.(*Profile)
		if p.UpdatedAt == pp.UpdatedAt() {
			return pp, nil
		}
	}

	params, err := s.store.GetProfile(p.Id)
	if err != nil {
		return nil, err
	}

	pp = newProfile(s, params)
	if err = pp.Login(); err != nil {

		return nil, err
	}

	s.profiles.Add(p.Id, pp)

	return pp, nil
}

func (s *server) fetchNewMessageInProfile(p *model.EmailProfileTask) {
	profile, err := s.GetProfile(p)
	if err != nil {
		wlog.Error(err.Error())

		if err = s.store.SetError(p.Id, err); err != nil {
			wlog.Error(err.Error())
		}
		return
	}

	emails := profile.Read()
	for _, email := range emails {
		s.consume <- NewConnection(profile, email)
	}
}
