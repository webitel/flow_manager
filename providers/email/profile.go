package email

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/smtp"
	"net/textproto"
	"strings"
	"sync"
	"time"

	"github.com/emersion/go-message/mail"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client"
	"github.com/jordan-wright/email"
	"github.com/webitel/flow_manager/model"
	"github.com/webitel/wlog"

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

	server *server
	sync.RWMutex
	client *client.Client

	mbox        *imap.MailboxStatus
	lastMessage time.Time
}

func newProfile(srv *server, params *model.EmailProfile) *Profile {
	return &Profile{
		Id:        params.Id,
		DomainId:  params.DomainId,
		updatedAt: params.UpdatedAt,
		server:    srv,
		login:     params.Login,
		password:  params.Password,
		smtpHost:  params.SmtpHost,
		smtpPort:  params.SmtpPort,
		imapHost:  params.ImapHost,
		imapPort:  params.ImapPort,
		Mailbox:   params.Mailbox,
		flowId:    params.FlowId,
		name:      params.Name,
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
	p.client, err = client.DialTLS(fmt.Sprintf("%s:%d", p.imapHost, p.imapPort), nil)
	if err != nil {
		return model.NewAppError("Email", "email.login.app_err", nil, err.Error(), http.StatusInternalServerError)
	}

	if err := p.client.Login(p.login, p.password); err != nil {
		return model.NewAppError("Email", "email.login.unauthorized", nil, err.Error(), http.StatusUnauthorized)
	}
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

	p.client.Close()
	p.client = nil
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

func (p *Profile) Read() []*model.Email {
	res := make([]*model.Email, 0)

	criteria := imap.NewSearchCriteria()
	criteria.WithoutFlags = []string{"\\Seen"}

	if err := p.selectMailBox(); err != nil {
		p.storeErr(err)
		return nil
	}

	uids, err := p.client.UidSearch(criteria)
	if err != nil {
		p.storeErr(model.NewAppError("Email", "email.mailbox.search.app_err", nil, err.Error(), http.StatusInternalServerError))
		return nil
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
	return res
}

func (p *Profile) Reply(parent *model.Email, data []byte) (*model.Email, *model.AppError) {
	id, err := generateMessageID()
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

	if err := e.Send(fmt.Sprintf("%s:%d", p.smtpHost, p.smtpPort), smtp.PlainAuth("", p.login, p.password, p.smtpHost)); err != nil {
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

		switch part.Header.(type) {
		case *mail.InlineHeader:
			ct := part.Header.Get("Content-Type")
			// This is the message's text (can be plain-text or HTML)
			b, _ := ioutil.ReadAll(part.Body)
			if strings.HasPrefix(ct, "text/html") {
				html = b
			} else if strings.HasPrefix(ct, "text/") {
				text = append(text, b...)
			}
		case *mail.AttachmentHeader:
			// This is an attachment
			// TODO
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
