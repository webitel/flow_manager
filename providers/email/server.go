package email

import (
	"fmt"
	"sync"
	"time"

	"golang.org/x/oauth2"

	"github.com/webitel/flow_manager/storage"

	"github.com/webitel/engine/discovery"
	"github.com/webitel/engine/utils"
	"github.com/webitel/flow_manager/model"
	"github.com/webitel/flow_manager/store"
	"github.com/webitel/wlog"
)

var (
	FetchProfileInterval = time.Second * 2
	SizeCache            = 1000
)

const (
	MailGmail   = "gmail"
	MailOutlook = "outlook"
)

type MailServer struct {
	store           store.EmailStore
	storage         *storage.Api
	profiles        *utils.Cache
	didFinishListen chan struct{}
	stopped         chan struct{}
	startOnce       sync.Once
	consume         chan model.Connection
	debug           bool
}

func New(storageApi *storage.Api, s store.EmailStore, debug bool) model.Server {
	return &MailServer{
		store:           s,
		profiles:        utils.NewLru(SizeCache),
		didFinishListen: make(chan struct{}),
		stopped:         make(chan struct{}),
		consume:         make(chan model.Connection),
		storage:         storageApi,
		debug:           debug,
	}
}

func (s *MailServer) Name() string {
	return "Email"
}

func (s *MailServer) Cluster(discovery discovery.ServiceDiscovery) *model.AppError {
	return nil
}

func (s *MailServer) Start() *model.AppError {
	s.startOnce.Do(func() {
		go s.listen()
	})
	return nil
}

func (s *MailServer) Stop() {
	close(s.didFinishListen)
	<-s.stopped
}

func (s *MailServer) Host() string {
	return "" //TODO
}

func (s *MailServer) Port() int {
	return 0
}

func (s *MailServer) Type() model.ConnectionType {
	return model.ConnectionTypeEmail
}

func (s *MailServer) Consume() <-chan model.Connection {
	return s.consume
}

func (s *MailServer) listen() {
	defer func() {
		wlog.Debug("stop listen email server...")
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

func (s *MailServer) GetProfile(id int, updatedAt int64) (*Profile, *model.AppError) {
	var pp *Profile
	profile, ok := s.profiles.Get(id)
	if ok {
		pp = profile.(*Profile)
		if updatedAt == pp.UpdatedAt() {
			return pp, nil
		}
	}

	params, err := s.store.GetProfile(id)
	if err != nil {
		return nil, err
	}

	pp = newProfile(s, params)
	//if err = pp.Login(); err != nil {
	//
	//	return nil, err
	//}

	s.profiles.Add(id, pp)

	return pp, nil
}

func (s *MailServer) fetchNewMessageInProfile(p *model.EmailProfileTask) {
	profile, err := s.GetProfile(p.Id, p.UpdatedAt)
	if err != nil {
		wlog.Error(err.Error())

		if err = s.store.SetError(p.Id, err); err != nil {
			wlog.Error(err.Error())
		}
		return
	}

	attempts := 0

retry:
	err = profile.Login()
	if err != nil {
		s.storeError(profile, err)
		wlog.Error(fmt.Sprintf("profile \"%s\", error: %s", profile, err.Error()))
	}

	var emails []*model.Email
	emails, err = profile.Read()
	if err != nil {
		if err.DetailedError == "Not logged in" {
			if attempts == 0 {
				attempts = attempts + 1
				goto retry
			}
		}
		wlog.Error(fmt.Sprintf("[%s] error: %s", profile, err.Error()))
		return
	}

	for _, email := range emails {
		if err = s.store.Save(profile.DomainId, email); err != nil {
			wlog.Error(fmt.Sprintf("%s, error: %s", profile, err.Error()))
			continue
		}
		s.consume <- NewConnection(profile, email)
	}
	err = profile.Logout()
	if err != nil {
		s.storeError(profile, err)
		wlog.Error(fmt.Sprintf("profile \"%s\", error: %s", profile, err.Error()))
	}
}

func (s *MailServer) storeError(p *Profile, err *model.AppError) {
	saveErr := s.store.SetError(p.Id, err)
	if saveErr != nil {
		wlog.Error(fmt.Sprintf("%s, error: %s", p, saveErr.Error()))
	}
}

func (s *MailServer) storeToken(p *Profile, token *oauth2.Token) {
	err := s.store.SetToken(p.Id, token)
	if err != nil {
		wlog.Error(fmt.Sprintf("profile_id=%d, store token error: %s", p.Id, err.Error()))
	}
}

func (s *MailServer) TestProfile(domainId int64, profileId int) *model.AppError {
	var profile *Profile
	updatedAt, err := s.store.GetProfileUpdatedAt(domainId, profileId)
	if err != nil {
		return err
	}

	if profile, err = s.GetProfile(profileId, updatedAt); err != nil {
		return err
	}

	if err = profile.Login(); err != nil {
		return err
	}

	if err = profile.Logout(); err != nil {
		return err
	}

	return nil
}
