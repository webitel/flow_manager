package flow

import (
	"context"
	"crypto/tls"
	"io"
	"net/http"

	"github.com/webitel/flow_manager/model"
	"github.com/webitel/wlog"
	"gopkg.in/gomail.v2"
)

type EmailArgs struct {
	Cc        []string `json:"cc"`
	From      string   `json:"from"`
	Message   string   `json:"message"`
	ReplyToId string   `json:"replyToId"`
	Type      string   `json:"type"`
	Smtp      struct {
		Auth struct {
			Password string `json:"password"`
			User     string `json:"user"`
		} `json:"auth"`
		Port     int    `json:"port"`
		Server   string `json:"server"`
		Tls      bool   `json:"tls"`
		Insecure bool   `json:"insecure"`
	} `json:"smtp"`
	Subject    string   `json:"subject"`
	To         []string `json:"to"`
	Async      bool     `json:"async"`
	Attachment struct {
		Files []model.File `json:"files"`
	} `json:"attachment"`
}

type GetEmailInfo struct {
	Email *struct {
		Id        *int64  `json:"id"`
		MessageId *string `json:"messageId"`
	}
	Set model.Variables
}

func (r *router) sendEmail(ctx context.Context, scope *Flow, conn model.Connection, args interface{}) (model.Response, *model.AppError) {
	var argv EmailArgs
	argv.Type = "text/html"

	if err := scope.Decode(args, &argv); err != nil {
		return nil, err
	}

	if argv.Message == "" {
		return nil, ErrorRequiredParameter("sendEmail", "message")
		// err
	}
	if argv.To == nil || len(argv.To) < 1 {
		return nil, ErrorRequiredParameter("sendEmail", "to")
	}
	if argv.Smtp.Server == "" {
		return nil, ErrorRequiredParameter("sendEmail", "server")
	}
	if argv.Smtp.Auth.User == "" {
		return nil, ErrorRequiredParameter("sendEmail", "user")
	}
	if argv.Smtp.Port == 0 {
		return nil, ErrorRequiredParameter("sendEmail", "port")
	}

	if argv.Async {
		go r.sendEmailFn(conn.DomainId(), argv)
		return ResponseOK, nil
	} else {
		return r.sendEmailFn(conn.DomainId(), argv)
	}
}

func (r *router) sendEmailFn(domainId int64, argv EmailArgs) (model.Response, *model.AppError) {

	mail := gomail.NewMessage()
	mail.SetHeader("To", argv.To...)

	if argv.From != "" {
		mail.SetHeader("From", argv.From)
	}

	if argv.Subject != "" {
		mail.SetHeader("Subject", argv.Subject)
	}

	if argv.ReplyToId != "" {
		mail.SetHeader("In-Reply-To", argv.ReplyToId[1:len(argv.ReplyToId)-1])
	}

	if len(argv.Attachment.Files) != 0 {
		ids := make([]int64, 0, len(argv.Attachment.Files))
		for _, v := range argv.Attachment.Files {
			ids = append(ids, int64(v.Id))
		}
		files, err := r.fm.GetFileMetadata(domainId, ids)
		if err != nil {
			wlog.Error(err.Error())
		} else {
			r.attachToMail(domainId, mail, files)
		}
	}

	mail.SetBody(argv.Type, argv.Message)

	dialer := gomail.NewDialer(argv.Smtp.Server, argv.Smtp.Port, argv.Smtp.Auth.User, argv.Smtp.Auth.Password)
	if argv.Smtp.Tls {
		dialer.TLSConfig = &tls.Config{InsecureSkipVerify: true}
	}

	if sErr := dialer.DialAndSend(mail); sErr != nil {
		return nil, model.NewAppError("Email", "flow.email.send.app_err", nil, sErr.Error(), http.StatusInternalServerError)
	}

	return model.CallResponseOK, nil
}

func (r *router) attachToMail(domainId int64, mail *gomail.Message, files []model.File) {
	attachFn := func(f model.File) func(w io.Writer) error {
		return func(w io.Writer) error {
			reader, err := r.fm.DownloadFile(domainId, int64(f.Id))
			if err != nil {
				wlog.Error(err.Error())
				return err
			}
			defer reader.Close()

			_, cErr := io.Copy(w, reader)
			return cErr
		}
	}

	for _, file := range files {
		mail.Attach(file.GetViewName(), gomail.SetCopyFunc(attachFn(file)))
	}
}

func (r *router) getEmail(ctx context.Context, scope *Flow, conn model.Connection, args interface{}) (model.Response, *model.AppError) {
	var argv GetEmailInfo
	var err *model.AppError
	if err = scope.Decode(args, &argv); err != nil {
		return nil, err
	}

	if argv.Email == nil {
		return nil, ErrorRequiredParameter("GetEmailInfo", "email")
	}

	if argv.Set == nil {
		return nil, ErrorRequiredParameter("GetEmailInfo", "set")
	}

	res, err := r.fm.GetEmailProperties(conn.DomainId(), argv.Email.Id, argv.Email.MessageId, argv.Set)
	if err != nil {
		return nil, err
	}

	return conn.Set(ctx, res)
}
