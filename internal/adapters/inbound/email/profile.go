package email

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/smtp"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client"
	"github.com/emersion/go-message/mail"
	"github.com/k3a/html2text"
	"golang.org/x/oauth2"
	"gopkg.in/gomail.v2"

	"github.com/webitel/wlog"

	emaildomain "github.com/webitel/flow_manager/internal/domain/email"
	"github.com/webitel/flow_manager/internal/domain/files"
	domstorage "github.com/webitel/flow_manager/internal/domain/storage"
	apperrs "github.com/webitel/flow_manager/internal/infrastructure/errors"
	"github.com/webitel/flow_manager/internal/infrastructure/utils"

	_ "github.com/emersion/go-message/charset"
)

type Profile struct {
	Id        int
	DomainId  int64
	updatedAt int64
	name      string
	login     string
	password  string
	Mailbox   string
	smtpHost  string
	smtpPort  int
	imapHost  string
	imapPort  int

	logged bool

	flowId int

	server *MailServer
	sync.RWMutex
	client *client.Client

	mbox        *imap.MailboxStatus
	lastMessage time.Time

	authMethod  string
	oauthConfig oauth2.Config
	token       *oauth2.Token
	Tls         bool
	log         *wlog.Logger
}

func newProfile(srv *MailServer, params *emaildomain.EmailProfile) *Profile {
	return &Profile{
		Id:          params.Id,
		DomainId:    params.DomainId,
		updatedAt:   params.UpdatedAt,
		server:      srv,
		login:       params.Login,
		password:    params.Password,
		smtpHost:    params.SmtpHost,
		smtpPort:    params.SmtpPort,
		imapHost:    params.ImapHost,
		imapPort:    params.ImapPort,
		Mailbox:     params.Mailbox,
		flowId:      params.FlowId,
		name:        params.Name,
		oauthConfig: params.OAuthConfig(),
		token:       params.Token,
		authMethod:  params.AuthType,
		Tls:         params.Tls(),
		log: srv.log.With(
			wlog.String("scope", "email profile"),
			wlog.Int("profile_id", params.Id),
			wlog.Int("schema_id", params.FlowId),
		),
	}
}

func (p *Profile) String() string {
	return fmt.Sprintf("%s <%s>", p.name, p.login)
}

func (p *Profile) Login() error {
	done := make(chan error)
	// TODO WTEL-4468
	go func() {
		done <- p.clientLogin()
	}()
	select {
	case err := <-done:
		return err
	case <-time.After(time.Minute):
		return apperrs.New(http.StatusInternalServerError, "Email: email.login.timeout: Timeout")
	}
}

func (p *Profile) clientLogin() error {
	p.Lock()
	defer p.Unlock()

	if p.logged && p.client != nil {
		return nil
	}

	p.logged = false
	var err error

	var tlsConfig *tls.Config

	if p.Tls {
		tlsConfig = &tls.Config{}
		tlsConfig.InsecureSkipVerify = true
	}

	dialer := new(net.Dialer)
	dialer.Timeout = time.Second * 20
	p.client, err = client.DialWithDialerTLS(dialer, fmt.Sprintf("%s:%d", p.imapHost, p.imapPort), tlsConfig)
	if err != nil {
		return fmt.Errorf("Email: email.dial.app_err: %w", err)
	}

	if p.server.debug {
		p.client.SetDebug(os.Stdout)
	}

	if p.authMethod == emaildomain.MailAuthTypeOAuth2 {
		var ok bool
		ok, err = p.client.SupportAuth(Xoauth2)
		if err != nil {
			return fmt.Errorf("Email: email.xoauth2.support: %w", err)
		}

		if !ok {
			return fmt.Errorf("Email: email.xoauth2.support: Not support")
		}

		if p.token == nil {
			return fmt.Errorf("Email: email.xoauth2.support: Not found token")
		}

		lastExpiry := p.token.Expiry

		ts := p.oauthConfig.TokenSource(context.Background(), p.token)
		newToken, err := ts.Token()
		if err != nil {
			return apperrs.New(http.StatusUnauthorized, fmt.Sprintf("Email: email.login.token: %s", err.Error()))
		}

		if !newToken.Expiry.Equal(lastExpiry) {
			p.storeToken(newToken)
		}

		p.token = newToken

		saslClient := NewXoauth2Client(p.login, newToken.AccessToken)

		err = p.client.Authenticate(saslClient)
		if err != nil {
			return apperrs.New(http.StatusUnauthorized, fmt.Sprintf("Email: email.login.unauthorized: %s", err.Error()))
		}
	} else {
		if err = p.client.Login(p.login, p.password); err != nil {
			return apperrs.New(http.StatusUnauthorized, fmt.Sprintf("Email: email.login.unauthorized: %s", err.Error()))
		}
	}
	p.log.Debug("logged in")
	p.logged = true
	return nil
}

func (p *Profile) Logout() error {
	p.Lock()
	defer p.Unlock()

	if !p.logged {
		return nil
	}

	err := p.client.Logout()
	if err != nil {
		return fmt.Errorf("Email: email.logout.app_err: %w", err)
	}
	p.logged = false
	p.client.Close()
	p.client = nil
	p.log.Debug("logged out")
	return nil
}

func (p *Profile) UpdatedAt() int64 {
	p.RLock()
	defer p.RUnlock()

	return p.updatedAt
}

func (p *Profile) selectMailBox() error {
	var err error
	p.mbox, err = p.client.Select(p.Mailbox, false)
	if err != nil {
		return fmt.Errorf("Email: email.mailbox.app_err: %w", err)
	}

	return nil
}

func (p *Profile) storeErr(err error) {
	p.server.storeError(p, err)
}

func (p *Profile) storeToken(token *oauth2.Token) {
	p.server.storeToken(p, token)
}

func (p *Profile) Read() ([]*emaildomain.Email, error) {
	if !p.logged {
		if err := p.Login(); err != nil {
			return nil, err
		}
	}
	res := make([]*emaildomain.Email, 0)

	criteria := imap.NewSearchCriteria()
	criteria.WithoutFlags = []string{"\\Seen"}

	if err := p.selectMailBox(); err != nil {
		p.storeErr(err)
		return nil, err
	}

	uids, err := p.client.UidSearch(criteria)
	if err != nil {
		appErr := fmt.Errorf("Email: email.mailbox.search.app_err: %w", err)
		p.storeErr(appErr)
		return nil, appErr
	}

	if len(uids) == 0 {
		return res, nil
	}

	seqSet := new(imap.SeqSet)
	seqSet.AddNum(uids...)
	section := &imap.BodySectionName{}
	items := []imap.FetchItem{imap.FetchEnvelope, imap.FetchFlags, imap.FetchInternalDate, section.FetchItem()}
	messages := make(chan *imap.Message)

	done := make(chan struct{})

	go func() {
		if err := p.client.UidFetch(seqSet, items, messages); err != nil {
			// log.Fatal(err) //TODO
		}
		close(done)
	}()

	for msg := range messages {
		e, err := p.parseMessage(msg, section)
		if err != nil {
			p.log.Err(err)
			continue
		}

		p.log.Debug("receive new email", wlog.Any("from", e.From))
		res = append(res, e)
	}

	<-done
	return res, nil
}

func (p *Profile) Reply(parent *emaildomain.Email, data []byte) (*emaildomain.Email, error) {
	id, err := emaildomain.GenerateMailID()
	if err != nil {
		return nil, fmt.Errorf("Email: email.reply.app_err: %w", err)
	}

	rr := &emaildomain.Email{
		Direction: "outbound", // FIXME
		MessageId: id,
		Subject:   parent.Subject,
		ProfileId: parent.ProfileId,
		From:      []string{p.login},
		To:        parent.From,
		Sender:    parent.Sender,
		ReplyTo:   parent.ReplyTo,
		InReplyTo: parent.MessageId,
		CC:        parent.CC,
		Body:      data,
		HtmlBody:  data,
	}

	mail := gomail.NewMessage()
	mail.SetHeader("Message-Id", rr.MessageId)
	mail.SetHeader("From", p.login)
	mail.SetHeader("To", rr.To...)

	if rr.Subject != "" {
		mail.SetHeader("Subject", rr.Subject)
	}

	if rr.InReplyTo != "" {
		mail.SetHeader("In-Reply-To", rr.InReplyTo)
	}

	if len(rr.CC) != 0 {
		mail.SetHeader("Cc", rr.CC...)
	}

	mail.SetBody("text/html", string(rr.HtmlBody))
	var dialer *gomail.Dialer

	if p.authMethod == emaildomain.MailAuthTypeOAuth2 {
		p.log.Debug("using OAuth2",
			wlog.String("from", p.login),
			wlog.String("smtpHost", p.smtpHost),
			wlog.Int("smtpPort", p.smtpPort),
		)
		dialer = gomail.NewDialer(p.smtpHost, p.smtpPort, p.login, "")
		if p.token != nil && p.token.AccessToken != "" {
			dialer.Auth = NewOAuth2Smtp(p.login, "Bearer", p.token.AccessToken)
		} else {
			p.log.Error("no token provided")
		}
	} else {
		dialer = gomail.NewDialer(p.smtpHost, p.smtpPort, p.login, p.password)
	}

	if p.Tls {
		dialer.TLSConfig = &tls.Config{InsecureSkipVerify: true}
	}

	err = dialer.DialAndSend(mail)
	if err != nil {
		return nil, fmt.Errorf("Email: email.reply.app_err: %w", err)
	}

	return rr, nil
}

func (p *Profile) parseMessage(msg *imap.Message, section *imap.BodySectionName) (*emaildomain.Email, error) {
	m := &emaildomain.Email{
		ProfileId: p.Id,
		Direction: "inbound", // TODO
	}

	if msg == nil {
		return nil, fmt.Errorf("Email: email.message.app_err: Server didn't returned message")
	}

	r := msg.GetBody(section)
	if r == nil {
		return nil, fmt.Errorf("Email: email.message.app_err: Server didn't returned message body")
	}

	m.Subject = msg.Envelope.Subject
	for _, v := range msg.Envelope.From {
		m.From = append(m.From, v.Address())
	}
	for _, v := range msg.Envelope.Sender {
		m.Sender = append(m.Sender, v.Address())
	}
	for _, v := range msg.Envelope.ReplyTo {
		m.ReplyTo = append(m.ReplyTo, v.Address())
	}
	for _, v := range msg.Envelope.To {
		m.To = append(m.To, v.Address())
	}
	for _, v := range msg.Envelope.Cc {
		m.CC = append(m.CC, v.Address())
	}
	m.InReplyTo = msg.Envelope.InReplyTo
	m.MessageId = msg.Envelope.MessageId

	// Create a new mail reader
	mr, err := mail.CreateReader(r)
	if err != nil {
		return nil, fmt.Errorf("Email: email.message.app_err: %w", err)
	}

	var text []byte
	var html []byte

	m.Cid = make(map[string]emaildomain.EmailCid)

	// Process each message's part
	var part *mail.Part
	for {
		part, err = mr.NextPart()
		if err == io.EOF {
			break
		} else if err != nil {
			p.log.Error(err.Error(), wlog.String("message-id", m.MessageId),
				wlog.Any("from", m.From),
			)
			break
		}

		if part == nil {
			p.log.Error("empty part", wlog.String("message-id", m.MessageId),
				wlog.Any("from", m.From),
			)
			break
		}

		switch h := part.Header.(type) {
		case *mail.InlineHeader:
			cid := h.Get("Content-ID")
			if cid != "" {
				var file domstorage.File
				cid = strings.Trim(cid, "<>")
				file, err = p.server.storage.Upload(context.TODO(), p.DomainId, m.MessageId, part.Body, domstorage.File{
					Name:     cid,
					MimeType: h.Get("Content-Type"),
					Channel:  domstorage.ChannelMail,
				})
				if err != nil {
					p.log.With(wlog.Any("from", m.From)).Error(err.Error(), wlog.Err(err))
					continue
				}
				m.Cid[cid] = emaildomain.EmailCid(file.Id)
			}
			ct := h.Get("Content-Type")
			// This is the message's text (can be plain-text or HTML)
			b, _ := ioutil.ReadAll(part.Body)
			if strings.HasPrefix(ct, "text/html") {
				html = b
			} else if strings.HasPrefix(ct, "text/") {
				text = append(text, b...)
			}
		case *mail.AttachmentHeader:
			var fileName string
			fileName, err = h.Filename()
			if err != nil {
				p.log.With(wlog.Any("from", m.From)).Err(err)
				continue
			}
			if fileName == "" {
				fileName = utils.NewId()
			}

			var file domstorage.File
			file, err = p.server.storage.Upload(context.TODO(), p.DomainId, m.MessageId, part.Body, domstorage.File{
				Name:     fileName,
				MimeType: h.Get("Content-Type"),
				Channel:  domstorage.ChannelMail,
			})
			if err != nil {
				p.log.With(wlog.Any("from", m.From)).Error(err.Error(), wlog.Err(err))
				continue
			}
			m.Attachments = append(m.Attachments, files.File{
				Id:       file.Id,
				Url:      file.Url,
				Name:     file.Name,
				Size:     file.Size,
				MimeType: file.MimeType,
			})
		}
	}

	if len(text) == 0 && len(html) != 0 {
		text = []byte(html2text.HTML2Text(string(html)))
	}

	if text != nil {
		m.Body = text
	} else {
		m.Body = html
	}
	m.HtmlBody = html

	return m, nil
}

type OAuth2Smtp struct {
	user, tokenType, token string
}

// Returns an AUTH that implements XOAUTH2 authentication
// user is your email username (normally your email address)
// tokenType is usually going to be "Bearer"
// token is your access_token generated by a tool like quickstart
func NewOAuth2Smtp(user, tokenType, token string) smtp.Auth {
	return &OAuth2Smtp{user, tokenType, token}
}

func (a *OAuth2Smtp) Start(server *smtp.ServerInfo) (string, []byte, error) {
	if !server.TLS {
		return "", nil, errors.New("unencrypted connection")
	}
	resp := []byte(fmt.Sprintf("user=%v\001auth=%v %v\001\001", a.user, "Bearer", a.token))
	return "XOAUTH2", resp, nil
}

func (a *OAuth2Smtp) Next(fromServer []byte, more bool) ([]byte, error) {
	if more {
		// We've already sent everything.
		return nil, fmt.Errorf("unexpected server challenge: %s", fromServer)
	}
	return nil, nil
}
