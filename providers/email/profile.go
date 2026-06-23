package email

import (
	"bytes"
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"mime"
	"net"
	"net/http"
	"net/smtp"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client"
	"github.com/emersion/go-message/charset"
	"github.com/emersion/go-message/mail"
	"github.com/k3a/html2text"
	"golang.org/x/oauth2"
	"gopkg.in/gomail.v2"

	"github.com/webitel/wlog"

	"github.com/webitel/flow_manager/model"
)

type Profile struct {
	sync.RWMutex

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
	client *client.Client

	mbox        *imap.MailboxStatus
	lastMessage time.Time

	authMethod  string
	oauthConfig oauth2.Config
	token       *oauth2.Token
	Tls         bool
	log         *wlog.Logger
	decoder     *mime.WordDecoder
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
		Tls:         params.Tls(),
		log: srv.log.With(
			wlog.String("scope", "email profile"),
			wlog.Int("profile_id", params.Id),
			wlog.Int("schema_id", params.FlowId),
		),
		decoder: &mime.WordDecoder{CharsetReader: charset.Reader},
	}
}

func (p *Profile) String() string { return fmt.Sprintf("%s <%s>", p.name, p.login) }

func (p *Profile) Login() *model.AppError {
	done := make(chan *model.AppError)
	// TODO WTEL-4468
	go func() {
		done <- p.clientLogin()
	}()

	select {
	case err := <-done:
		if err != nil {
			return err
		}

		return nil
	case <-time.After(time.Minute):
		return model.NewAppError("Email", "email.login.timeout", nil, "Timeout", 500)
	}
}

func (p *Profile) clientLogin() *model.AppError {
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

	p.log.Debug("logged in")
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
	p.log.Debug("logged out")

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

func (p *Profile) storeErr(err *model.AppError)   { p.server.storeError(p, err) }
func (p *Profile) storeToken(token *oauth2.Token) { p.server.storeToken(p, token) }

func (p *Profile) Read() ([]*model.Email, *model.AppError) {
	if !p.logged {
		if err := p.Login(); err != nil {
			return nil, err
		}
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
			p.log.Error("fetching client UID", wlog.Err(err))
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

func (p *Profile) Reply(parent *model.Email, data []byte) (*model.Email, *model.AppError) {
	id, err := model.GenerateMailID()
	if err != nil {
		return nil, model.NewAppError("Email", "email.reply.app_err", nil, err.Error(), http.StatusInternalServerError)
	}

	rr := &model.Email{
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

	if p.authMethod == model.MailAuthTypeOAuth2 {
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
		return nil, model.NewAppError("Email", "email.reply.app_err", nil, err.Error(), http.StatusInternalServerError)
	}

	return rr, nil
}

func prepareBodySection(msg *imap.Message, section *imap.BodySectionName) (imap.Literal, *model.AppError) {
	if msg == nil {
		return nil, model.NewAppError("Email", "email.message.app_err", nil, "Server didn't returned message", http.StatusInternalServerError)
	}

	if msg.Envelope == nil {
		return nil, model.NewAppError("Email", "email.message.app_err", nil, "Message does`nt contain envelope", http.StatusInternalServerError)
	}

	r := msg.GetBody(section)
	if r == nil {
		return nil, model.NewAppError("Email", "email.message.app_err", nil, "Server didn't returned message body", http.StatusInternalServerError)
	}

	return r, nil
}

func decodeAddress(address []*imap.Address, decoder *mime.WordDecoder) []string {
	strAddrs := make([]string, 0, len(address))
	for _, a := range address {
		s := a.Address()
		if parser, err := decoder.DecodeHeader(s); err == nil {
			strAddrs = append(strAddrs, parser)

			continue
		}

		strAddrs = append(strAddrs, s)
	}

	return strAddrs
}

func decodeText(s string, decoder *mime.WordDecoder) string {
	if d, err := decoder.DecodeHeader(s); err == nil {
		return d
	}

	return s
}

func createEmailFromEnvelope(profileID int, envelope *imap.Envelope, decoder *mime.WordDecoder) *model.Email {
	return &model.Email{
		Direction: model.MailDirectionInbound,
		MessageId: envelope.MessageId,
		Subject:   decodeText(envelope.Subject, decoder),
		ProfileId: profileID,
		From:      decodeAddress(envelope.From, decoder),
		To:        decodeAddress(envelope.To, decoder),
		Sender:    decodeAddress(envelope.Sender, decoder),
		ReplyTo:   decodeAddress(envelope.ReplyTo, decoder),
		InReplyTo: decodeText(envelope.InReplyTo, decoder),
		CC:        decodeAddress(envelope.Cc, decoder),
		Cid:       make(map[string]model.EmailCid),
	}
}

func (p *Profile) parseMessage(msg *imap.Message, section *imap.BodySectionName) (*model.Email, *model.AppError) {
	bodySection, werr := prepareBodySection(msg, section)
	if werr != nil {
		return nil, werr
	}

	receivedMail := createEmailFromEnvelope(p.Id, msg.Envelope, p.decoder)
	logger := p.log.With(wlog.String("message_id", receivedMail.MessageId), wlog.Any("from", receivedMail.From))

	mr, err := mail.CreateReader(bodySection)
	if err != nil {
		return nil, model.NewAppError("Email", "email.message.app_err", nil, err.Error(), http.StatusInternalServerError)
	}
	defer mr.Close()

	it := NewMailIterator(mr, logger)

	//	TODO: remove after creating high level context that will pass to this func,
	// 	with span id for OTEL integrity
	handlersCtx := context.TODO()

	var text, html []byte

	for it.Next() {
		part := it.Part()

		switch h := part.Header.(type) {
		case *mail.InlineHeader:
			bodyText, bodyHtml, err := p.inlineHeaderHandler(handlersCtx, h, part.Body, receivedMail)
			if err != nil {
				logger.Error("processing inline header, continue to next part", wlog.Err(err))

				continue
			}

			if len(bodyText) > 0 {
				text = append(text, bodyText...)
			}

			if len(bodyHtml) > 0 {
				html = bodyHtml
			}
		case *mail.AttachmentHeader:
			if err := p.attachmentHeaderHandler(handlersCtx, h, part.Body, receivedMail); err != nil {
				logger.Error("processing attachment header of mail, continue to next part", wlog.Err(err))

				continue
			}
		}
	}

	if err := it.Err(); err != nil {
		return nil, err
	}

	text = resolveMailMessageText(text, html)

	receivedMail.SetBody(text, html).SetHTMLBody(html)

	return receivedMail, nil
}

func (p *Profile) inlineHeaderHandler(ctx context.Context, inlineHeader *mail.InlineHeader, body io.Reader, receivedMail *model.Email) ([]byte, []byte, error) {
	b, err := io.ReadAll(body)
	if err != nil {
		return nil, nil, err
	}

	contentType := inlineHeader.Get("Content-Type")

	var text, html []byte
	if strings.HasPrefix(contentType, "text/html") {
		html = b
	} else if strings.HasPrefix(contentType, "text/") {
		text = append(text, b...)
	}

	cid := strings.Trim(inlineHeader.Get("Content-ID"), "<>")

	if cid == "" {
		return text, html, nil
	}

	file, err := p.server.storage.Upload(
		ctx,
		p.DomainId,
		receivedMail.MessageId,
		bytes.NewReader(b),
		model.File{Name: cid, MimeType: contentType, Channel: model.FileChannelMail},
	)
	if err != nil {
		return nil, nil, err
	}

	receivedMail.AddCID(cid, file.Id)

	return text, html, nil
}

func (p *Profile) attachmentHeaderHandler(ctx context.Context, attachmentHeader *mail.AttachmentHeader, body io.Reader, receivedMail *model.Email) error {
	fileName, err := attachmentHeader.Filename()
	if err != nil {
		return err
	}

	if fileName == "" {
		fileName = model.NewId()
	}

	file, err := p.server.storage.Upload(
		ctx,
		p.DomainId,
		receivedMail.MessageId,
		body,
		model.File{Name: fileName, MimeType: attachmentHeader.Get("Content-Type"), Channel: model.FileChannelMail},
	)
	if err != nil {
		return err
	}

	receivedMail.AddAttachment(file)

	return nil
}

func resolveMailMessageText(text, html []byte) []byte {
	if len(text) == 0 && len(html) != 0 {
		return []byte(html2text.HTML2Text(string(html)))
	}

	return text
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

	resp := fmt.Appendf(nil, "user=%v\001auth=%v %v\001\001", a.user, "Bearer", a.token)

	return "XOAUTH2", resp, nil
}

func (a *OAuth2Smtp) Next(fromServer []byte, more bool) ([]byte, error) {
	if more {
		// We've already sent everything.
		return nil, fmt.Errorf("unexpected server challenge: %s", fromServer)
	}

	return nil, nil
}
