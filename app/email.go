package app

import (
	"net/http"

	"github.com/webitel/flow_manager/model"
	"github.com/webitel/flow_manager/internal/adapters/inbound/email"
)

func (f *FlowManager) ReplyEmail(conn model.EmailConnection, text string) *model.AppError {
	email, err := conn.Reply(text)
	if err != nil {
		return err
	}

	if storeErr := f.Store.Email().Save(conn.DomainId(), email); storeErr != nil {
		return model.NewAppError("ReplyEmail", "store.email.save", nil, storeErr.Error(), http.StatusInternalServerError)
	}
	return nil
}

func (f *FlowManager) MailServer() *email.MailServer {
	return f.mailServer.(*email.MailServer)
}
