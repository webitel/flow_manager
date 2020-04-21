package email

import (
	"fmt"
	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client"
	"github.com/emersion/go-message/mail"
	"github.com/jordan-wright/email"
	"github.com/webitel/flow_manager/model"
	"github.com/webitel/wlog"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/smtp"
	"net/textproto"
	"sync"
	"time"
)

type Profile struct {
	Id       int
	DomainId int64
	Addr     string
	login    string
	password string
	Mailbox  string
	smtpPort int
	imapPort int

	server *server
	sync.Mutex
	client *client.Client

	mbox        *imap.MailboxStatus
	lastMessage time.Time
}

func newProfile(srv *server, params *model.EmailProfile) *Profile {
	return &Profile{
		Id:       params.Id,
		DomainId: params.DomainId,
		server:   srv,
		Addr:     params.Host,
		login:    params.Login,
		password: params.Password,
		smtpPort: params.SmtpPort,
		imapPort: params.ImapPort,
		Mailbox:  params.Mailbox,
	}
}

func (p *Profile) Login() *model.AppError {
	var err error
	//TODO port
	p.client, err = client.DialTLS(fmt.Sprintf("%s:%d", p.Addr, p.imapPort), nil)
	if err != nil {
		return model.NewAppError("Email", "email.login.app_err", nil, err.Error(), http.StatusInternalServerError)
	}

	if err := p.client.Login(p.login, p.password); err != nil {
		return model.NewAppError("Email", "email.login.unauthorized", nil, err.Error(), http.StatusUnauthorized)
	}

	return nil
}

func (p *Profile) selectMailBox() *model.AppError {
	var err error
	p.mbox, err = p.client.Select(p.Mailbox, false)
	if err != nil {
		return model.NewAppError("Email", "email.mailbox.app_err", nil, err.Error(), http.StatusInternalServerError)
	}

	return nil
}

func (p *Profile) Read() []*model.Email {
	res := make([]*model.Email, 0)

	criteria := imap.NewSearchCriteria()
	criteria.WithoutFlags = []string{"\\Seen"}

	if err := p.selectMailBox(); err != nil {
		wlog.Error(err.Error())
		return nil
	}

	uids, err := p.client.UidSearch(criteria)
	if err != nil {
		log.Println(err)
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

	for message := range messages {
		e, err := p.parseMessage(message, section)
		if err != nil {
			wlog.Error(err.Error())
			continue
		}

		if err = p.server.store.Save(p.DomainId, e); err != nil {
			//TODO
			wlog.Error(err.Error())
			continue
		}

		wlog.Debug(fmt.Sprintf("receive new email from %v					", e.From))
		res = append(res, e)
	}

	<-done
	return res
}

func (p *Profile) Reply(parent *model.Email, data []byte) *model.AppError {
	id, err := generateMessageID()
	if err != nil {
		return model.NewAppError("Email", "email.reply.app_err", nil, err.Error(), http.StatusInternalServerError)
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
		Body:      data, //[]byte("<h1>Fancy HTML is supported, too!</h1>"),
	}

	if err := p.server.store.Save(p.DomainId, rr); err != nil {
		return err
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

	if err := e.Send(fmt.Sprintf("%s:%d", p.Addr, p.smtpPort), smtp.PlainAuth("", p.login, p.password, p.Addr)); err != nil {
		return model.NewAppError("Email", "email.reply.app_err", nil, err.Error(), http.StatusInternalServerError)
	}

	return nil
}

func (p *Profile) parseMessage(message *imap.Message, section *imap.BodySectionName) (*model.Email, *model.AppError) {
	email := &model.Email{
		ProfileId: p.Id,
		Direction: "inbound", //TODO
	}

	if message == nil {
		return nil, model.NewAppError("Email", "email.message.app_err", nil, "Server didn't returned message",
			http.StatusInternalServerError)
	}

	r := message.GetBody(section)
	if r == nil {
		return nil, model.NewAppError("Email", "email.message.app_err", nil, "Server didn't returned message body",
			http.StatusInternalServerError)
	}

	// Create a new mail reader
	mr, err := mail.CreateReader(r)
	if err != nil {
		return nil, model.NewAppError("Email", "email.message.app_err", nil, err.Error(),
			http.StatusInternalServerError)
	}

	email.Subject = message.Envelope.Subject
	for _, v := range message.Envelope.From {
		email.From = append(email.From, v.Address())
	}
	for _, v := range message.Envelope.Sender {
		email.Sender = append(email.Sender, v.Address())
	}
	for _, v := range message.Envelope.ReplyTo {
		email.ReplyTo = append(email.ReplyTo, v.Address())
	}
	for _, v := range message.Envelope.To {
		email.To = append(email.To, v.Address())
	}
	for _, v := range message.Envelope.Cc {
		email.CC = append(email.CC, v.Address())
	}
	email.InReplyTo = message.Envelope.InReplyTo
	email.MessageId = message.Envelope.MessageId

	// Process each message's part
	for {
		part, err := mr.NextPart()
		if err == io.EOF {
			break
		} else if err != nil {
			log.Fatal(err)
		}

		switch part.Header.(type) {
		case *mail.InlineHeader:
			// This is the message's text (can be plain-text or HTML)
			b, _ := ioutil.ReadAll(part.Body)
			email.Body = append(email.Body, b...)
		}
	}

	return email, nil
}
