package flow

import (
	"context"
	"crypto/tls"
	"github.com/webitel/flow_manager/model"
	"gopkg.in/gomail.v2"
	"net/http"
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
	Subject string   `json:"subject"`
	To      []string `json:"to"`
	Async   bool     `json:"async"`
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
		go r.sendEmailFn(argv)
		return ResponseOK, nil
	} else {
		return r.sendEmailFn(argv)
	}
}

func (r *router) sendEmailFn(argv EmailArgs) (model.Response, *model.AppError) {

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
