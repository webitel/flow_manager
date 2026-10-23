package app

import (
	"context"

	"github.com/webitel/flow_manager/model"
	"github.com/webitel/flow_manager/providers/email"
)

func (f *FlowManager) LastBridgedEmail(ctx context.Context, domainId int64, number, hours string, queueIds []int, mapRes model.Variables) (model.Variables, *model.AppError) {
	return f.Store.Email().LastBridged(ctx, domainId, number, hours, queueIds, mapRes)
}

func (f *FlowManager) GetEmailProperties(domainId int64, id *int64, messageId *string, mapRes model.Variables) (model.Variables, *model.AppError) {
	return f.Store.Email().GerProperties(domainId, id, messageId, mapRes)
}

func (f *FlowManager) ReplyEmail(conn model.EmailConnection, text string) *model.AppError {
	email, err := conn.Reply(text)
	if err != nil {
		return err
	}

	return f.Store.Email().Save(conn.DomainId(), email)
}

func (f *FlowManager) MailServer() *email.MailServer {
	return f.mailServer.(*email.MailServer)
}
