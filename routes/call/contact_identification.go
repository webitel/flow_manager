package call

import (
	"context"
	"encoding/json"
	"strconv"
	"time"

	"github.com/webitel/flow_manager/contactid"
	"github.com/webitel/flow_manager/flow"
	"github.com/webitel/flow_manager/model"
)

// contactIdentification is the "Contact identification" node for calls. The
// caller's number is a verified observation; the node's variables are
// unverified criteria. Candidates are kept in the call's params, the schema
// gets only the summary, and a single verified candidate is bound when the
// call has no contact yet. A failed run leaves the variable empty.
func (r *Router) contactIdentification(ctx context.Context, scope *flow.Flow, call model.Call, args any) (model.Response, *model.AppError) {
	var argv model.ContactIdentificationArgs
	if err := r.Decode(scope, args, &argv); err != nil {
		return model.CallResponseError, err
	}

	if argv.SetVar == "" {
		return model.CallResponseError, model.ErrorRequiredParameter(model.ContactIdentificationApp, "setVar")
	}

	lastN, ok := argv.PhoneLastDigitsOrWhole()
	if !ok {
		call.Log().Warn("contact identification: phoneLastDigits must be 4..15, the whole number is matched")
	}

	searchCtx, cancel := context.WithTimeout(ctx, nodeTimeout)
	defer cancel()

	res, _, err := identifyContact(searchCtx, r.fm, call, contactid.Request{
		Criteria:        argv.Criteria(),
		PhoneLastDigits: lastN,
		PrimaryOnly:     argv.PrimaryPhonesOnly(),
	}, false)
	if err != nil {
		call.Log().Error("contact identification failed: " + err.Error())

		return call.Set(ctx, model.Variables{argv.SetVar: ""})
	}

	// The search context may be spent; setting the variable uses the node's.
	return call.Set(ctx, model.Variables{argv.SetVar: model.ToJson(res)})
}

const (
	// nodeTimeout bounds the CRM and database work of one node run.
	nodeTimeout = 5 * time.Second
	// autoLinkTimeout bounds the auto-link that runs before every call schema.
	autoLinkTimeout = 2 * time.Second
	// saveAttempts covers a call row the ringing event has not inserted yet.
	saveAttempts   = 3
	saveRetryDelay = 200 * time.Millisecond
	// saveRetryTimeout bounds the detached retries of one save.
	saveRetryTimeout = 2 * time.Second
)

// callContacts is what call identification needs from the flow manager.
type callContacts interface {
	contactid.Searcher
	CallContactCandidates(domainID int64, callID string) ([]byte, *model.AppError)
	CallSetContactCandidates(domainID int64, callID string, candidates, duplicate []byte, onlyIfAbsent bool) (bool, *model.AppError)
	CallSetContactIDIfEmpty(domainID int64, callID string, contactID int64) (set, found bool, appErr *model.AppError)
	UserNotification(n model.Notification)
}

// identifyContact runs one identification for the call: searches, appends to
// the stored candidates and binds a single verified one if nothing is bound.
// autoLink is the pass before the schema: with no match it touches nothing,
// and it never waits for the call row the ringing event inserts. It reports
// whether it bound a contact. When the CRM refused every criterion, the
// caller-number matches are still kept and bound, and the error is returned.
func identifyContact(ctx context.Context, fm callContacts, call model.Call, req contactid.Request, autoLink bool) (*model.ContactIdentificationResult, bool, *model.AppError) {
	req.DomainID = call.DomainId()
	req.CallerNumber = callerNumber(call)

	matches, skipped, searchErr := contactid.Search(ctx, fm, req)
	if skipped > 0 && searchErr == nil {
		call.Log().Warn("contact identification: " + strconv.Itoa(skipped) + " criteria skipped, their CRM search failed")
	}

	if searchErr != nil && len(matches) == 0 {
		return nil, false, searchErr
	}

	if autoLink && len(matches) == 0 {
		return &model.ContactIdentificationResult{}, false, nil
	}

	var existing []contactid.Candidate

	// Keeping the list is best-effort: a failure here must not cost the call
	// its contact, which the auto-link always bound. When the stored list
	// could not be read, this run only fills an empty one: it never replaces
	// candidates it did not see.
	unread := false

	raw, err := fm.CallContactCandidates(call.DomainId(), call.Id())
	if err != nil {
		call.Log().Error("contact identification: read candidates: " + err.Error())

		unread = true
	} else if len(raw) > 0 {
		if e := json.Unmarshal(raw, &existing); e != nil {
			call.Log().Error("contact identification: stored candidates are unreadable: " + e.Error())

			unread = true
		}
	}

	all, changed := contactid.Merge(existing, matches, model.GetMillis())
	if changed {
		saveCandidates(ctx, fm, call, all, autoLink, unread)
	}

	// A contact already on the call (chosen by an operator, set by a schema or
	// an earlier run) is never replaced.
	onCall, bound := "", false
	if id := call.GetContactId(); id != 0 {
		onCall = strconv.Itoa(id)
	} else if id, ok := contactid.SingleVerified(all); ok && bindContact(fm, call, id) {
		onCall, bound = id, true
	}

	if searchErr != nil {
		return nil, bound, searchErr
	}

	res := contactid.Summary(all, onCall)

	return &res, bound, nil
}

// saveCandidates writes the list and the duplicate record of two or more
// verified candidates. The first attempt runs inline; while the call row the
// ringing event inserts does not exist yet, the retries run detached so the
// call is never held. A detached retry only fills an empty list: it never
// replaces what a later run stored.
func saveCandidates(ctx context.Context, fm callContacts, call model.Call, all []contactid.Candidate, detachRetries, onlyIfAbsent bool) {
	data, e := json.Marshal(all)
	if e != nil {
		call.Log().Error("contact identification: encode candidates: " + e.Error())

		return
	}

	var dup []byte
	if d := contactid.Duplicates(all); d != nil {
		if dup, e = json.Marshal(d); e != nil {
			call.Log().Error("contact identification: encode duplicate: " + e.Error())
		}
	}

	write := func(onlyIfAbsent bool) func() (bool, *model.AppError) {
		return func() (bool, *model.AppError) {
			return fm.CallSetContactCandidates(call.DomainId(), call.Id(), data, dup, onlyIfAbsent)
		}
	}

	found, err := write(onlyIfAbsent)()
	if err != nil {
		call.Log().Error("contact identification: save candidates: " + err.Error())

		return
	}

	if found {
		return
	}

	if !detachRetries {
		// The run waits for its own retries: no later run of this call can
		// store a list meanwhile, so they write as the first attempt did.
		retryUntilRow(ctx, call, "candidates not kept", write(onlyIfAbsent))

		return
	}

	go func() {
		ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), saveRetryTimeout)
		defer cancel()

		retryUntilRow(ctx, call, "candidates not kept", write(true))
	}()
}

// bindContact sets the call's contact and tells the calling user's client.
func bindContact(fm callContacts, call model.Call, id string) bool {
	contactID, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		call.Log().Error("contact identification: bad contact id " + id)

		return false
	}

	// Only a call without a contact is bound: one set meanwhile (by an
	// operator or a schema) is never replaced.
	// appErr keeps its concrete type: assigned to err (an error) a nil
	// *AppError would compare non-nil.
	set, found, appErr := fm.CallSetContactIDIfEmpty(call.DomainId(), call.Id(), contactID)
	if appErr != nil {
		call.Log().Error("contact identification: set contact: " + appErr.Error())

		return false
	}

	// The row exists and already has a contact: it stays.
	if found && !set {
		return false
	}

	// No row yet (the ringing event has not inserted it): bind the call as the
	// auto-link always did, and write the row once it exists, in the background.
	if !found {
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), saveRetryTimeout)
			defer cancel()

			retryUntilRow(ctx, call, "contact kept on the channel only", func() (bool, *model.AppError) {
				_, found, appErr := fm.CallSetContactIDIfEmpty(call.DomainId(), call.Id(), contactID)

				return found, appErr
			})
		}()
	}

	_, _ = call.Set(call.Context(), model.Variables{"wbt_contact_id": id})

	notifyContact(fm, call, &contactID)

	return true
}

// retryUntilRow repeats write, saveRetryDelay apart, while the call row it
// writes to does not exist yet (write reports found = false), up to
// saveAttempts in all counting the caller's first attempt.
func retryUntilRow(ctx context.Context, call model.Call, lost string, write func() (found bool, err *model.AppError)) {
	for attempt := 2; attempt <= saveAttempts; attempt++ {
		select {
		case <-ctx.Done():
			return
		case <-time.After(saveRetryDelay):
		}

		found, err := write()
		if err != nil {
			call.Log().Error("contact identification: write to the call: " + err.Error())

			return
		}

		if found {
			return
		}
	}

	call.Log().Error("contact identification: call row not found, " + lost)
}

func notifyContact(fm callContacts, call model.Call, contactID *int64) {
	userIDStr, _ := call.Get("sip_h_X-Webitel-User-Id")

	userID, _ := strconv.Atoi(userIDStr)
	if userID <= 0 {
		return
	}

	fm.UserNotification(model.Notification{
		DomainId:  call.DomainId(),
		Action:    "set_contact",
		CreatedAt: model.GetMillis(),
		ForUsers:  []int64{int64(userID)},
		Body: map[string]any{
			"id":         call.Id(),
			"contact_id": contactID,
			"channel":    model.CallExchange,
		},
	})
}

// callerNumber is the client's side of the call: the destination of an
// outbound call, the caller of any other.
func callerNumber(call model.Call) string {
	if call.Direction() == model.CallDirectionOutbound {
		return call.Destination()
	}

	if from := call.From(); from != nil && from.GetNumber() != nil {
		return *from.GetNumber()
	}

	return ""
}
