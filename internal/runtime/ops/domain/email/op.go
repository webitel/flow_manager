// Package email provides the native sendEmail op.
package email

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"mime"

	"github.com/webitel/wlog"
	"gopkg.in/gomail.v2"

	"github.com/webitel/flow_manager/internal/runtime/ops"
	"github.com/webitel/flow_manager/model"
	emailprovider "github.com/webitel/flow_manager/providers/email"
)

// EmailDeps is the narrow interface required by the sendEmail op.
type EmailDeps interface {
	SmtpSettings(domainId int64, search *model.SearchEntity) (*model.SmtSettings, error)
	SmtpSettingsOAuthToken(settings *model.SmtSettings) (string, error)
	GetFileMetadata(domainId int64, ids []int64) ([]model.File, error)
	DownloadFile(domainId int64, id int64) (io.ReadCloser, error)
	SaveEmail(domainId int64, email *model.Email) error
}

// Register adds sendEmail to reg.
func Register(reg *ops.Registry, deps EmailDeps) {
	reg.Register("sendEmail", &sendEmailOp{deps: deps})
}

// ── sendEmail ─────────────────────────────────────────────────────────────────

type sendEmailOp struct{ deps EmailDeps }

func (o *sendEmailOp) Kind() ops.OpKind { return ops.OpKindSync }

type emailArgs struct {
	Cc         []string            `json:"cc"`
	From       string              `json:"from"`
	Message    string              `json:"message"`
	ReplyToId  string              `json:"replyToId"`
	Type       string              `json:"type"`
	Profile    *model.SearchEntity `json:"profile"`
	Smtp       model.SmtSettings   `json:"smtp"`
	Subject    string              `json:"subject"`
	To         []string            `json:"to"`
	ContactIds []int64             `json:"contactIds"`
	OwnerId    *int64              `json:"ownerId"`
	Async      bool                `json:"async"`
	Attachment struct {
		Files []model.File `json:"files"`
	} `json:"attachment"`
	RetryCount int               `json:"retryCount"`
	Store      bool              `json:"store"`
	Set        map[string]string `json:"set"`
}

func (o *sendEmailOp) Execute(ctx context.Context, in ops.OpInput) (ops.OpOutput, error) {
	argv := emailArgs{Type: "text/html"}
	if err := ops.DecodeArgs(in, &argv); err != nil {
		return ops.OpOutput{}, fmt.Errorf("sendEmail: %w", err)
	}

	if argv.Message == "" {
		return ops.OpOutput{}, fmt.Errorf("sendEmail: message is required")
	}
	if len(argv.To) == 0 {
		return ops.OpOutput{}, fmt.Errorf("sendEmail: to is required")
	}

	if argv.Profile != nil && (argv.Profile.Id != nil || argv.Profile.Name != nil) {
		if settings, err := o.deps.SmtpSettings(in.DomainID, argv.Profile); err == nil {
			argv.Smtp = *settings
		}
	}

	if argv.Async {
		go func() {
			if err := o.sendFn(context.Background(), in.DomainID, argv, nil); err != nil {
				wlog.Error(fmt.Sprintf("sendEmail async: %v", err))
			}
		}()
		return ops.OpOutput{}, nil
	}

	setVars := make(map[string]string)
	err := o.sendFn(ctx, in.DomainID, argv, setVars)
	if err != nil {
		if k, ok := argv.Set["error"]; ok {
			setVars[k] = err.Error()
		}
		return ops.OpOutput{SetVars: setVars}, err
	}
	return ops.OpOutput{SetVars: setVars}, nil
}

// sendFn executes the full email send (SMTP dialing, optional store, deferred var
// setting into setVars). setVars may be nil (async path — no variable output).
func (o *sendEmailOp) sendFn(ctx context.Context, domainID int64, argv emailArgs, setVars map[string]string) error {
	mail := gomail.NewMessage()
	mail.SetHeader("To", argv.To...)

	if argv.From == "" && argv.Smtp.Auth.User != "" {
		argv.From = argv.Smtp.Auth.User
	}
	if argv.From != "" {
		mail.SetHeader("From", argv.From)
	}
	if argv.Subject != "" {
		mail.SetHeader("Subject", argv.Subject)
	}
	if argv.ReplyToId != "" {
		mail.SetHeader("In-Reply-To", argv.ReplyToId[1:len(argv.ReplyToId)-1])
	}
	if len(argv.Cc) != 0 {
		mail.SetHeader("Cc", argv.Cc...)
	}

	var attachedFiles []model.File
	if len(argv.Attachment.Files) != 0 {
		ids := make([]int64, 0, len(argv.Attachment.Files))
		for _, f := range argv.Attachment.Files {
			ids = append(ids, int64(f.Id))
		}
		if meta, err := o.deps.GetFileMetadata(domainID, ids); err != nil {
			wlog.Error(fmt.Sprintf("sendEmail: get file metadata: %v", err))
		} else {
			attachedFiles = meta
			o.attachToMail(domainID, mail, meta)
		}
	}

	mail.SetBody(argv.Type, argv.Message)

	var dialer *gomail.Dialer
	if argv.Smtp.AuthType == model.MailAuthTypeOAuth2 {
		token, err := o.deps.SmtpSettingsOAuthToken(&argv.Smtp)
		if err != nil {
			return fmt.Errorf("sendEmail: oauth token: %w", err)
		}
		dialer = gomail.NewDialer(argv.Smtp.Server, argv.Smtp.Port, argv.Smtp.Auth.User, "")
		dialer.Auth = emailprovider.NewOAuth2Smtp(argv.Smtp.Auth.User, "Bearer", token)
	} else {
		dialer = gomail.NewDialer(argv.Smtp.Server, argv.Smtp.Port, argv.Smtp.Auth.User, argv.Smtp.Auth.Password)
	}

	if argv.Smtp.Tls {
		dialer.TLSConfig = &tls.Config{InsecureSkipVerify: true} //nolint:gosec
	}

	id, genErr := model.GenerateMailID()
	if genErr != nil {
		return fmt.Errorf("sendEmail: generate mail id: %w", genErr)
	}
	mail.SetHeader("Message-Id", id)

	if setVars != nil {
		if k, ok := argv.Set["message_id"]; ok {
			setVars[k] = id
		}
	}

	retries := argv.RetryCount
	for {
		if sErr := dialer.DialAndSend(mail); sErr != nil {
			if retries > 0 {
				retries--
				wlog.Error(fmt.Sprintf("sendEmail: dial: %v (retrying)", sErr))
				continue
			}
			return fmt.Errorf("sendEmail: %w", sErr)
		}
		break
	}

	if argv.Store && argv.Smtp.Id > 0 {
		rr := &model.Email{
			Direction:   "outbound",
			MessageId:   id,
			Subject:     argv.Subject,
			ProfileId:   argv.Smtp.Id,
			From:        []string{argv.From},
			To:          argv.To,
			Sender:      []string{argv.From},
			InReplyTo:   argv.ReplyToId,
			CC:          argv.Cc,
			Body:        []byte(argv.Message),
			HtmlBody:    []byte(argv.Message),
			Attachments: attachedFiles,
		}
		if argv.OwnerId != nil && *argv.OwnerId > 0 {
			rr.OwnerId = argv.OwnerId
		}
		if len(argv.ContactIds) != 0 {
			rr.ContactIds = argv.ContactIds
		}
		if argv.ReplyToId != "" {
			rr.InReplyTo = argv.ReplyToId[1 : len(argv.ReplyToId)-1]
		}

		if saveErr := o.deps.SaveEmail(domainID, rr); saveErr != nil {
			return fmt.Errorf("sendEmail: save: %w", saveErr)
		}

		if setVars != nil {
			if k, ok := argv.Set["id"]; ok {
				setVars[k] = fmt.Sprintf("%d", rr.Id)
			}
		}
	}

	return nil
}

func (o *sendEmailOp) attachToMail(domainID int64, mail *gomail.Message, files []model.File) {
	attachFn := func(f model.File) func(w io.Writer) error {
		return func(w io.Writer) error {
			reader, err := o.deps.DownloadFile(domainID, int64(f.Id))
			if err != nil {
				wlog.Error(fmt.Sprintf("sendEmail: download file %d: %v", f.Id, err))
				return err
			}
			defer reader.Close()
			_, cErr := io.Copy(w, reader)
			return cErr
		}
	}
	for _, file := range files {
		mail.Attach(mime.QEncoding.Encode("UTF-8", file.GetViewName()), gomail.SetCopyFunc(attachFn(file)))
	}
}

// ensure interface
var _ ops.Op = (*sendEmailOp)(nil)
var _ ops.Documenter = (*sendEmailOp)(nil)
