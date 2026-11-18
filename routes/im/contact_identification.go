package im

import (
	"context"

	"github.com/webitel/wlog"

	"github.com/webitel/flow_manager/flow"
	"github.com/webitel/flow_manager/model"
)

// contactIdentification is the "Contact identification" node for chats:
// im-thread searches the CRM and keeps the candidates on the thread;
// the schema gets only the summary. A failed run leaves the variable empty, so
// the next node can branch on it, and the schema goes on.
func (r *Router) contactIdentification(ctx context.Context, scope *flow.Flow, conv Dialog, args any) (model.Response, *model.AppError) {
	var argv model.ContactIdentificationArgs
	if err := r.Decode(scope, args, &argv); err != nil {
		return model.CallResponseError, err
	}

	if argv.SetVar == "" {
		return model.CallResponseError, model.ErrorRequiredParameter(model.ContactIdentificationApp, "setVar")
	}

	lastN, ok := argv.PhoneLastDigitsOrWhole()
	if !ok {
		wlog.Warn("contact identification: phoneLastDigits must be 4..15, the whole number is matched", wlog.String("thread_id", conv.ThreadId()))
	}

	// IdentifyContact bounds the call to im-thread itself; ctx stays usable
	// for setting the variable afterwards.
	res, err := conv.IdentifyContact(ctx, model.ContactIdentificationRequest{
		ThreadID:        argv.ConversationID,
		Criteria:        argv.Criteria(),
		PhoneLastDigits: lastN,
		PrimaryOnly:     argv.PrimaryPhonesOnly(),
	})
	if err != nil {
		wlog.Error("contact identification failed", wlog.String("thread_id", conv.ThreadId()), wlog.Err(err))

		return conv.Set(ctx, model.Variables{argv.SetVar: ""})
	}

	return conv.Set(ctx, model.Variables{argv.SetVar: model.ToJson(res)})
}
