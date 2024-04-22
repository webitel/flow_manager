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
	"net/textproto"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/webitel/wlog"

	"github.com/emersion/go-imap"

	"golang.org/x/oauth2"

	"github.com/emersion/go-imap/client"
	"github.com/jordan-wright/email"
	"github.com/webitel/flow_manager/model"

	_ "github.com/emersion/go-message/charset"
	"github.com/emersion/go-message/mail"
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
}

func newProfile(srv *MailServer, params *model.EmailProfile) *Profile {
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
	}
}

func (p *Profile) String() string {
	return fmt.Sprintf("%s <%s>", p.name, p.login)
}

func (p *Profile) Login() *model.AppError {
	p.Lock()
	defer p.Unlock()

	if p.logged && p.client != nil {
		return nil
	}

	p.logged = false
	var err error

	tlsConfig := &tls.Config{}

	// todo
	if p.Tls {
		tlsConfig.InsecureSkipVerify = true
	}

	dialer := new(net.Dialer)
	dialer.Timeout = time.Second * 20
	p.client, err = client.DialWithDialerTLS(dialer, fmt.Sprintf("%s:%d", p.imapHost, p.imapPort), tlsConfig)
	if err != nil {
		return model.NewAppError("Email", "email.dial.app_err", nil, err.Error(), http.StatusInternalServerError)
	}

	if p.server.debug {
		p.client.SetDebug(os.Stdout)
	}

	if p.authMethod == model.MailAuthTypeOAuth2 {
		var ok bool
		ok, err = p.client.SupportAuth(Xoauth2)
		if err != nil {
			return model.NewAppError("Email", "email.xoauth2.support", nil, err.Error(), http.StatusInternalServerError)
		}

		if !ok {
			return model.NewAppError("Email", "email.xoauth2.support", nil, "Not support", http.StatusInternalServerError)
		}

		if p.token == nil {
			return model.NewAppError("Email", "email.xoauth2.support", nil, "Not found token", http.StatusInternalServerError)
		}

		lastExpiry := p.token.Expiry

		ts := p.oauthConfig.TokenSource(context.Background(), p.token)
		newToken, err := ts.Token()
		if err != nil {
			return model.NewAppError("Email", "email.login.token", nil, err.Error(), http.StatusUnauthorized)
		}

		if !newToken.Expiry.Equal(lastExpiry) {
			p.storeToken(newToken)
		}

		p.token = newToken

		saslClient := NewXoauth2Client(p.login, newToken.AccessToken)

		err = p.client.Authenticate(saslClient)
		if err != nil {
			return model.NewAppError("Email", "email.login.unauthorized", nil, err.Error(), http.StatusUnauthorized)
		}
	} else {
		if err = p.client.Login(p.login, p.password); err != nil {
			return model.NewAppError("Email", "email.login.unauthorized", nil, err.Error(), http.StatusUnauthorized)
		}
	}
	wlog.Debug(fmt.Sprintf("profile %s logged in", p.name))
	p.logged = true
	return nil
}

func (p *Profile) Logout() *model.AppError {
	p.Lock()
	defer p.Unlock()

	if !p.logged {
		return nil
	}

	err := p.client.Logout()
	if err != nil {
		return model.NewAppError("Email", "email.logout.app_err", nil, err.Error(), http.StatusInternalServerError)
	}
	p.logged = false
	p.client.Close()
	p.client = nil
	wlog.Debug(fmt.Sprintf("profile %s logged out", p.name))
	return nil
}

func (p *Profile) UpdatedAt() int64 {
	p.RLock()
	defer p.RUnlock()

	return p.updatedAt
}

func (p *Profile) selectMailBox() *model.AppError {
	var err error
	p.mbox, err = p.client.Select(p.Mailbox, false)
	if err != nil {
		return model.NewAppError("Email", "email.mailbox.app_err", nil, err.Error(), http.StatusInternalServerError)
	}

	return nil
}

func (p *Profile) storeErr(err *model.AppError) {
	p.server.storeError(p, err)
}

func (p *Profile) storeToken(token *oauth2.Token) {
	p.server.storeToken(p, token)
}

func (p *Profile) Read() ([]*model.Email, *model.AppError) {
	if !p.logged {
		if err := p.Login(); err != nil {
			return nil, err
		}
		//return nil, model.NewAppError("Email", "email.mailbox.not_logged", nil, "Profile not logged", http.StatusInternalServerError)
	}
	res := make([]*model.Email, 0)

	criteria := imap.NewSearchCriteria()
	criteria.WithoutFlags = []string{"\\Seen"}

	if err := p.selectMailBox(); err != nil {
		p.storeErr(err)
		return nil, err
	}

	uids, err := p.client.UidSearch(criteria)
	if err != nil {
		appErr := model.NewAppError("Email", "email.mailbox.search.app_err", nil, err.Error(), http.StatusInternalServerError)
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
			//log.Fatal(err) //TODO
		}
		close(done)
	}()

	for msg := range messages {
		e, err := p.parseMessage(msg, section)
		if err != nil {
			wlog.Error(fmt.Sprintf("%s, error: %s", p, err.Error()))
			continue
		}

		wlog.Debug(fmt.Sprintf("receive new email from %v", e.From))
		res = append(res, e)
	}

	<-done
	return res, nil
}

func (p *Profile) Reply(parent *model.Email, data []byte) (*model.Email, *model.AppError) {
	id, err := model.GenerateMailID()
	if err != nil {
		return nil, model.NewAppError("Email", "email.reply.app_err", nil, err.Error(), http.StatusInternalServerError)
	}

	rr := &model.Email{
		Direction: "outbound", //FIXME
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

	e := &email.Email{
		From:    p.login,
		To:      rr.To,
		Cc:      rr.CC,
		Subject: rr.Subject,
		//Text:    []byte("Text Body is, of course, supported!"),
		HTML: rr.Body,
		Headers: textproto.MIMEHeader{
			"In-Reply-To": []string{rr.InReplyTo},
			"Message-Id":  []string{rr.MessageId},
		},
	}

	var auth smtp.Auth
	if p.authMethod == model.MailAuthTypeOAuth2 {
		//  Authentication unsuccessful, SmtpClientAuthentication is disabled for the Tenant.
		auth = NewOAuth2Smtp(p.login, "Bearer", p.token.AccessToken)
	} else {
		auth = smtp.PlainAuth("", p.login, p.password, p.smtpHost)
	}

	if err := e.Send(fmt.Sprintf("%s:%d", p.smtpHost, p.smtpPort), auth); err != nil {
		return nil, model.NewAppError("Email", "email.reply.app_err", nil, err.Error(), http.StatusInternalServerError)
	}

	return rr, nil
}

func (p *Profile) parseMessage(msg *imap.Message, section *imap.BodySectionName) (*model.Email, *model.AppError) {
	m := &model.Email{
		ProfileId: p.Id,
		Direction: "inbound", //TODO
	}

	if msg == nil {
		return nil, model.NewAppError("Email", "email.message.app_err", nil, "Server didn't returned message",
			http.StatusInternalServerError)
	}

	r := msg.GetBody(section)
	if r == nil {
		return nil, model.NewAppError("Email", "email.message.app_err", nil, "Server didn't returned message body",
			http.StatusInternalServerError)
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
		return nil, model.NewAppError("Email", "email.message.app_err", nil, err.Error(),
			http.StatusInternalServerError)
	}

	var text []byte
	var html []byte

	// Process each message's part
	var part *mail.Part
	for {
		part, err = mr.NextPart()
		if err == io.EOF {
			break
		} else if err != nil {

		}

		switch h := part.Header.(type) {
		case *mail.InlineHeader:
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
				wlog.Error(fmt.Sprintf("email [%s] error: %s", m.From, err.Error()))
				continue
			}
			if fileName == "" {
				fileName = model.NewId()
			}

			var file model.File
			file, err = p.server.storage.Upload(context.TODO(), p.DomainId, m.MessageId, part.Body, model.File{
				Name:     fileName,
				MimeType: h.Get("Content-Type"),
			})
			if err != nil {
				wlog.Error(fmt.Sprintf("email [%s] error: %s", m.From, err.Error()))
				continue
			}
			m.Attachments = append(m.Attachments, file)
		}
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
