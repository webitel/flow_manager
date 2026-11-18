package call

import (
	"context"
	"encoding/json"
	"maps"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/webitel/wlog"

	"github.com/webitel/flow_manager/contactid"
	"github.com/webitel/flow_manager/gen/contacts"
	"github.com/webitel/flow_manager/model"
)

// testLog discards the identification logs.
var testLog = wlog.NewLogger(&wlog.LoggerConfiguration{})

type ifEmptyAnswer struct{ set, found bool }

type saveCall struct {
	onlyIfAbsent bool
	candidates   []contactid.Candidate
}

// fakeContacts is the flow manager side of call identification: a CRM with
// phone rows, the call's stored candidates and its contact.
type fakeContacts struct {
	mu sync.Mutex

	phones   []*contacts.Contact
	errByQin map[string]*model.AppError
	terms    []string

	stored  []byte
	readErr *model.AppError
	reads   int

	saveFound []bool
	saves     []saveCall

	ifEmpty []ifEmptyAnswer
	sets    chan int64

	notes []model.Notification
}

func (f *fakeContacts) SearchContactsNA(_ context.Context, req *contacts.SearchContactsNARequest) (*contacts.ContactList, *model.AppError) {
	f.mu.Lock()
	defer f.mu.Unlock()

	qin := req.GetQin()[0]
	f.terms = append(f.terms, qin+":"+req.GetQ())

	if err := f.errByQin[qin]; err != nil {
		return nil, err
	}

	if qin != "phones" {
		return &contacts.ContactList{}, nil
	}

	return &contacts.ContactList{Data: f.phones}, nil
}

func (f *fakeContacts) CallContactCandidates(int64, string) ([]byte, *model.AppError) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.reads++

	return f.stored, f.readErr
}

func (f *fakeContacts) CallSetContactCandidates(_ int64, _ string, candidates, _ []byte, onlyIfAbsent bool) (bool, *model.AppError) {
	f.mu.Lock()
	defer f.mu.Unlock()

	var all []contactid.Candidate

	_ = json.Unmarshal(candidates, &all)

	f.saves = append(f.saves, saveCall{onlyIfAbsent: onlyIfAbsent, candidates: all})

	found := true
	if len(f.saveFound) > 0 {
		found, f.saveFound = f.saveFound[0], f.saveFound[1:]
	}

	return found, nil
}

func (f *fakeContacts) CallSetContactIDIfEmpty(_ int64, _ string, contactID int64) (bool, bool, *model.AppError) {
	f.mu.Lock()
	answer := ifEmptyAnswer{set: true, found: true}

	if len(f.ifEmpty) > 0 {
		answer, f.ifEmpty = f.ifEmpty[0], f.ifEmpty[1:]
	}
	f.mu.Unlock()

	if f.sets != nil {
		f.sets <- contactID
	}

	return answer.set, answer.found, nil
}

func (f *fakeContacts) UserNotification(n model.Notification) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.notes = append(f.notes, n)
}

// fakeCall is a call with only what identification reads; any other method
// panics on the nil embedded interface.
type fakeCall struct {
	model.Call

	mu        sync.Mutex
	contactID int
	direction model.CallDirection
	from, to  string
	vars      model.Variables
}

func (c *fakeCall) DomainId() int64                { return 1 }
func (c *fakeCall) Id() string                     { return "call-1" }
func (c *fakeCall) Direction() model.CallDirection { return c.direction }
func (c *fakeCall) Destination() string            { return c.to }
func (c *fakeCall) From() *model.CallEndpoint      { return &model.CallEndpoint{Number: c.from} }
func (c *fakeCall) GetContactId() int              { return c.contactID }
func (c *fakeCall) Log() *wlog.Logger              { return testLog }
func (c *fakeCall) Context() context.Context       { return context.Background() }

func (c *fakeCall) Get(key string) (string, bool) {
	if key == "sip_h_X-Webitel-User-Id" {
		return "7", true
	}

	return "", false
}

func (c *fakeCall) Set(_ context.Context, vars model.Variables) (model.Response, *model.AppError) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.vars == nil {
		c.vars = model.Variables{}
	}

	maps.Copy(c.vars, vars)

	return model.CallResponseOK, nil
}

func (c *fakeCall) contactVar() any {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.vars["wbt_contact_id"]
}

func phoneRow(id, number string) *contacts.Contact {
	return &contacts.Contact{Id: id, Phones: &contacts.PhoneList{Data: []*contacts.PhoneNumber{{Number: number, Primary: true}}}}
}

func storedList(t *testing.T, cs ...contactid.Candidate) []byte {
	t.Helper()

	b, err := json.Marshal(cs)
	if err != nil {
		t.Fatal(err)
	}

	return b
}

func contactIDs(cs []contactid.Candidate) []string {
	out := make([]string, 0, len(cs))
	for _, c := range cs {
		out = append(out, c.ContactID)
	}

	return out
}

const caller = "+380000000001"

func TestIdentifyContact_Binding(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		contactID int
		ifEmpty   []ifEmptyAnswer
		wantBound bool
		wantSets  int
		wantID    string
	}{
		{name: "a contact already on the call is never replaced", contactID: 5, wantSets: 0},
		{name: "a contact set by a concurrent writer stays", ifEmpty: []ifEmptyAnswer{{set: false, found: true}}, wantSets: 1},
		{name: "a call without a contact is bound", wantBound: true, wantSets: 1, wantID: "10"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			fm := &fakeContacts{phones: []*contacts.Contact{phoneRow("10", caller)}, ifEmpty: tt.ifEmpty, sets: make(chan int64, 4)}
			call := &fakeCall{contactID: tt.contactID, from: caller}

			res, bound, err := identifyContact(context.Background(), fm, call, contactid.Request{}, false)
			if err != nil {
				t.Fatal(err)
			}

			if bound != tt.wantBound || res.ContactID != tt.wantID || res.FoundCount != 1 {
				t.Fatalf("bound = %v, result = %+v; want bound %v, contact_id %q", bound, res, tt.wantBound, tt.wantID)
			}

			if len(fm.sets) != tt.wantSets {
				t.Fatalf("conditional contact writes = %d, want %d", len(fm.sets), tt.wantSets)
			}

			if bound {
				if call.contactVar() != "10" || len(fm.notes) != 1 {
					t.Fatalf("wbt_contact_id = %v, notifications = %d; want 10 and one", call.contactVar(), len(fm.notes))
				}
			} else if call.contactVar() != nil || len(fm.notes) != 0 {
				t.Fatalf("not bound, but wbt_contact_id = %v, notifications = %d", call.contactVar(), len(fm.notes))
			}
		})
	}
}

// The ringing event has not inserted the call row yet: the call is bound at
// once and the row is written in the background once it exists.
func TestIdentifyContact_MissingRowIsWrittenLater(t *testing.T) {
	t.Parallel()

	fm := &fakeContacts{
		phones:  []*contacts.Contact{phoneRow("10", caller)},
		ifEmpty: []ifEmptyAnswer{{set: false, found: false}, {set: true, found: true}},
		sets:    make(chan int64, 4),
	}
	call := &fakeCall{from: caller}

	_, bound, err := identifyContact(context.Background(), fm, call, contactid.Request{}, false)
	if err != nil || !bound || call.contactVar() != "10" {
		t.Fatalf("bound = %v, err = %v, wbt_contact_id = %v; want bound to 10", bound, err, call.contactVar())
	}

	for want := range 2 {
		select {
		case id := <-fm.sets:
			if id != 10 {
				t.Fatalf("write %d binds %d, want 10", want+1, id)
			}
		case <-time.After(saveRetryTimeout):
			t.Fatalf("write %d of the contact never came", want+1)
		}
	}
}

func TestIdentifyContact_AutoLinkWithoutMatchTouchesNothing(t *testing.T) {
	t.Parallel()

	fm := &fakeContacts{sets: make(chan int64, 1)}

	res, bound, err := identifyContact(context.Background(), fm, &fakeCall{from: caller}, contactid.Request{}, true)
	if err != nil || bound || res.FoundCount != 0 {
		t.Fatalf("res = %+v, bound = %v, err = %v", res, bound, err)
	}

	if fm.reads != 0 || len(fm.saves) != 0 || len(fm.sets) != 0 {
		t.Fatalf("reads = %d, saves = %d, sets = %d; want none", fm.reads, len(fm.saves), len(fm.sets))
	}
}

func TestIdentifyContact_ClientNumber(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		direction model.CallDirection
		want      string
	}{
		{name: "inbound: the caller", direction: model.CallDirectionInbound, want: "phones:000000001"},
		{name: "outbound: the destination", direction: model.CallDirectionOutbound, want: "phones:000000002"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			fm := &fakeContacts{}

			call := &fakeCall{direction: tt.direction, from: "+380000000001", to: "+380000000002"}
			if _, _, err := identifyContact(context.Background(), fm, call, contactid.Request{}, true); err != nil {
				t.Fatal(err)
			}

			if !slices.Equal(fm.terms, []string{tt.want}) {
				t.Fatalf("CRM terms = %v, want %v", fm.terms, []string{tt.want})
			}
		})
	}
}

func TestIdentifyContact_SavesCandidates(t *testing.T) {
	t.Parallel()

	earlier := contactid.Candidate{
		ContactID: "20", Trust: model.IdentificationTrustUnverified, CreatedAt: 1,
		Reasons: []contactid.Reason{{Type: "email", Trust: model.IdentificationTrustUnverified, Origin: "flow"}},
	}

	tests := []struct {
		name      string
		stored    []byte
		readErr   *model.AppError
		saveFound []bool
		wantSaves []saveCall
	}{
		{
			name:      "a read list is merged and written over",
			stored:    storedList(t, earlier),
			wantSaves: []saveCall{{onlyIfAbsent: false, candidates: []contactid.Candidate{{ContactID: "10"}, earlier}}},
		},
		{
			name:      "an unread list is only filled when empty",
			readErr:   model.NewAppError("test", "test", nil, "db down", 500),
			wantSaves: []saveCall{{onlyIfAbsent: true, candidates: []contactid.Candidate{{ContactID: "10"}}}},
		},
		{
			name:      "an unreadable list is only filled when empty",
			stored:    []byte("{"),
			wantSaves: []saveCall{{onlyIfAbsent: true, candidates: []contactid.Candidate{{ContactID: "10"}}}},
		},
		{
			name:      "a missing row is retried inline by the node, written like the first attempt",
			saveFound: []bool{false, true},
			wantSaves: []saveCall{
				{onlyIfAbsent: false, candidates: []contactid.Candidate{{ContactID: "10"}}},
				{onlyIfAbsent: false, candidates: []contactid.Candidate{{ContactID: "10"}}},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			fm := &fakeContacts{phones: []*contacts.Contact{phoneRow("10", caller)}, stored: tt.stored, readErr: tt.readErr, saveFound: tt.saveFound}
			if _, _, err := identifyContact(context.Background(), fm, &fakeCall{contactID: 5, from: caller}, contactid.Request{}, false); err != nil {
				t.Fatal(err)
			}

			if len(fm.saves) != len(tt.wantSaves) {
				t.Fatalf("saves = %d, want %d", len(fm.saves), len(tt.wantSaves))
			}

			for i, want := range tt.wantSaves {
				got := fm.saves[i]
				if got.onlyIfAbsent != want.onlyIfAbsent || !slices.Equal(contactIDs(got.candidates), contactIDs(want.candidates)) {
					t.Fatalf("save %d = onlyIfAbsent %v, %v; want %v, %v", i, got.onlyIfAbsent, contactIDs(got.candidates), want.onlyIfAbsent, contactIDs(want.candidates))
				}
			}
		})
	}
}

// A CRM refusing every criterion fails the node, but the caller's own number
// is still kept and bound.
func TestIdentifyContact_CriteriaDownKeepsTheCaller(t *testing.T) {
	t.Parallel()

	fm := &fakeContacts{
		phones:   []*contacts.Contact{phoneRow("10", caller)},
		errByQin: map[string]*model.AppError{"emails": model.NewAppError("test", "test", nil, "down", 503)},
		sets:     make(chan int64, 1),
	}
	call := &fakeCall{from: caller}

	res, bound, err := identifyContact(context.Background(), fm, call, contactid.Request{Criteria: []model.ContactIdentificationCriterion{
		{Type: model.IdentificationCriterionEmail, Value: "qa@example.com"},
	}}, false)
	if err == nil || res != nil {
		t.Fatalf("res = %+v, err = %v; want the CRM error", res, err)
	}

	if !bound || call.contactVar() != "10" || len(fm.saves) != 1 {
		t.Fatalf("bound = %v, wbt_contact_id = %v, saves = %d; want the caller kept and bound", bound, call.contactVar(), len(fm.saves))
	}
}
