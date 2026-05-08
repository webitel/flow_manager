package app

import (
	"net/http"

	"github.com/webitel/flow_manager/model"
	"github.com/webitel/flow_manager/providers/email"
)

func (f *FlowManager) GetEmailProperties(domainId int64, id *int64, messageId *string, mapRes model.Variables) (model.Variables, *model.AppError) {
	vars, err := f.Store.Email().GerProperties(domainId, id, messageId, mapRes)
	if err != nil {
		return nil, model.NewAppError("GetEmailProperties", "store.email.get_properties", nil, err.Error(), http.StatusInternalServerError)
	}
	return vars, nil
}

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

func (f *FlowManager) SaveEmail(domainId int64, email *model.Email) error {
	return f.Store.Email().Save(domainId, email)
}

func (f *FlowManager) MailServer() *email.MailServer {
	return f.mailServer.(*email.MailServer)
}
