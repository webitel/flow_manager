package email

import (
	"context"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/sync/singleflight"

	"github.com/webitel/wlog"

	emaildomain "github.com/webitel/flow_manager/internal/domain/email"
	"github.com/webitel/flow_manager/internal/domain/flow"
	domstorage "github.com/webitel/flow_manager/internal/domain/storage"
	"github.com/webitel/flow_manager/internal/infrastructure/cache"
	"github.com/webitel/flow_manager/internal/infrastructure/discovery"
	"github.com/webitel/flow_manager/internal/storage"
)

var (
	FetchProfileInterval = time.Second * 10
	SizeCache            = 1000
	profileSGroup        singleflight.Group
)

type MailServer struct {
	store           storage.EmailStore
	storage         StorageApi
	profiles        *cache.LRUCache
	didFinishListen chan struct{}
	stopped         chan struct{}
	startOnce       sync.Once
	consume         chan flow.Connection
	running         map[int]bool
	debug           bool
	sync.RWMutex
	log *wlog.Logger
}

type StorageApi interface {
	Upload(ctx context.Context, domainId int64, uuid string, sFile io.Reader, metadata domstorage.File) (domstorage.File, error)
}

func New(storageApi StorageApi, s storage.EmailStore, debug bool) flow.Server {
	return &MailServer{
		store:           s,
		profiles:        cache.NewLru(SizeCache),
		didFinishListen: make(chan struct{}),
		stopped:         make(chan struct{}),
		consume:         make(chan flow.Connection),
		storage:         storageApi,
		running:         make(map[int]bool),
		debug:           debug,
		log:             wlog.GlobalLogger(),
	}
}

func (s *MailServer) Name() string {
	return "Email"
}

func (s *MailServer) Cluster(discovery discovery.ServiceDiscovery) error {
	return nil
}

func (s *MailServer) Start() error {
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
	return "" // TODO
}

func (s *MailServer) Port() int {
	return 0
}

func (s *MailServer) Type() flow.ConnectionType {
	return flow.ConnectionTypeEmail
}

func (s *MailServer) Consume() <-chan flow.Connection {
	return s.consume
}

func (s *MailServer) startRunning(id int) {
	s.Lock()
	s.running[id] = true
	s.Unlock()
}

func (s *MailServer) stopRunning(id int) {
	s.Lock()
	delete(s.running, id)
	s.Unlock()
}

func (s *MailServer) hasRunning(id int) bool {
	s.RLock()
	_, ok := s.running[id]
	s.RUnlock()
	return ok
}

func (s *MailServer) listen() {
	defer func() {
		s.log.Debug("stop listen email server...")
		close(s.stopped)
	}()

	s.log.Debug("start listen emails")
	for {
		select {
		case <-s.didFinishListen:
			return
		case <-time.After(FetchProfileInterval):
			tasks, err := s.store.ProfileTaskFetch("")
			if err != nil {
				s.log.Error(err.Error())
				time.Sleep(time.Second * 5)
			} else {
				for _, v := range tasks {
					if !s.hasRunning(v.Id) {
						s.startRunning(v.Id)
						go func(p *emaildomain.EmailProfileTask) {
							s.fetchNewMessageInProfile(p)
							s.stopRunning(p.Id)
						}(v)
					}
				}
			}
		}
	}
}

func (s *MailServer) GetProfile(id int, updatedAt int64) (*Profile, error) {
	var pp *Profile
	profile, ok := s.profiles.Get(id)
	if ok {
		pp = profile.(*Profile)
		if updatedAt == pp.UpdatedAt() {
			return pp, nil
		}
	}

	v, doErr, shared := profileSGroup.Do(fmt.Sprintf("%d-%d", id, updatedAt), func() (any, error) {
		params, err := s.store.GetProfile(id)
		if err != nil {
			return nil, err
		}

		return newProfile(s, params), nil
	})

	if doErr != nil {
		return nil, doErr
	}

	pp = v.(*Profile)

	if !shared {
		s.profiles.Add(id, pp)
	}

	return pp, nil
}

func (s *MailServer) fetchNewMessageInProfile(p *emaildomain.EmailProfileTask) {
	profile, err := s.GetProfile(p.Id, p.UpdatedAt)
	if err != nil {
		s.storeError(profile, err)
		s.log.Error(fmt.Sprintf("profile \"%s\", error: %s", profile, err.Error()))
		return
	}

	attempts := 0

retry:
	err = profile.Login()
	if err != nil {
		s.storeError(profile, err)
		s.log.Error(fmt.Sprintf("profile \"%s\", error: %s", profile, err.Error()))
	}

	var emails []*emaildomain.Email
	emails, err = profile.Read()
	if err != nil {
		if strings.Contains(err.Error(), "Not logged in") {
			if attempts == 0 {
				attempts = attempts + 1
				goto retry
			}
		}
		s.log.Error(fmt.Sprintf("[%s] error: %s", profile, err.Error()))
		return
	}

	for _, email := range emails {
		if storeErr := s.store.Save(profile.DomainId, email); storeErr != nil {
			s.log.Err(storeErr)
			continue
		}
		s.consume <- NewConnection(s, PKey{
			Id:        profile.Id,
			UpdatedAt: p.UpdatedAt,
			FlowId:    profile.flowId,
			DomainId:  profile.DomainId,
		}, email)
	}
	err = profile.Logout()
	if err != nil {
		s.storeError(profile, err)
		s.log.Err(err)
	}
}

func (s *MailServer) storeError(p *Profile, err error) {
	saveErr := s.store.SetError(p.Id, err)
	if saveErr != nil {
		s.log.Err(saveErr)
	}
	s.profiles.Remove(p.Id)
}

func (s *MailServer) storeToken(p *Profile, token *oauth2.Token) {
	err := s.store.SetToken(p.Id, token)
	if err != nil {
		s.log.Err(err)
	}
}

func (s *MailServer) TestProfile(domainId int64, profileId int) error {
	updatedAt, storeErr := s.store.GetProfileUpdatedAt(domainId, profileId)
	if storeErr != nil {
		return fmt.Errorf("TestProfile: store.email.get_profile_updated_at: %w", storeErr)
	}

	profile, err := s.GetProfile(profileId, updatedAt)
	if err != nil {
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
