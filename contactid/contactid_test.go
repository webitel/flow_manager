package contactid

import (
	"context"
	"slices"
	"strconv"
	"strings"
	"testing"

	"google.golang.org/protobuf/types/known/structpb"

	"github.com/webitel/flow_manager/gen/contacts"
	"github.com/webitel/flow_manager/model"
)

// fakeCRM answers per qin, page by page, and records every request.
type fakeCRM struct {
	byQin map[string][]*contacts.Contact
	err   *model.AppError
	// errByQin fails the searches of one qin only.
	errByQin map[string]*model.AppError
	reqs     []*contacts.SearchContactsNARequest
}

func (f *fakeCRM) SearchContactsNA(_ context.Context, req *contacts.SearchContactsNARequest) (*contacts.ContactList, *model.AppError) {
	f.reqs = append(f.reqs, req)

	if f.err != nil {
		return nil, f.err
	}

	if err := f.errByQin[req.GetQin()[0]]; err != nil {
		return nil, err
	}

	rows := f.byQin[req.GetQin()[0]]
	from := min(int((req.GetPage()-1)*req.GetSize()), len(rows))
	to := min(from+int(req.GetSize()), len(rows))

	return &contacts.ContactList{Data: rows[from:to], Next: to < len(rows)}, nil
}

func phoneContact(id string, numbers ...string) *contacts.Contact {
	c := &contacts.Contact{Id: id, Phones: &contacts.PhoneList{}}
	for i, n := range numbers {
		c.Phones.Data = append(c.Phones.Data, &contacts.PhoneNumber{Number: n, Primary: i == 0})
	}

	return c
}

func phone(v string) model.ContactIdentificationCriterion {
	return model.ContactIdentificationCriterion{Type: model.IdentificationCriterionPhone, Value: v}
}

func ids(ms []Match) []string {
	out := make([]string, 0, len(ms))
	for _, m := range ms {
		out = append(out, m.ContactID+":"+m.Reason.Type+":"+m.Reason.Trust)
	}

	return out
}

func TestSearch(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		crm   map[string][]*contacts.Contact
		req   Request
		want  []string
		terms []string
	}{
		{
			name:  "caller number is verified and trusted as the CRM matched it",
			crm:   map[string][]*contacts.Contact{"phones": {phoneContact("1", "011 000 00 01")}},
			req:   Request{CallerNumber: "+380110000001"},
			want:  []string{"1:caller_number:verified"},
			terms: []string{"110000001"},
		},
		{
			name: "caller number: substring hits of the CRM are re-checked",
			crm: map[string][]*contacts.Contact{"phones": {
				phoneContact("1", "+380110000001"),
				phoneContact("2", "3801100000019"),
				phoneContact("3", "0001"),
			}},
			req:   Request{CallerNumber: "0110000001"},
			want:  []string{"1:caller_number:verified"},
			terms: []string{"110000001"},
		},
		{
			name:  "short internal numbers match when equal",
			crm:   map[string][]*contacts.Contact{"phones": {phoneContact("1", "1001")}},
			req:   Request{CallerNumber: "1001"},
			want:  []string{"1:caller_number:verified"},
			terms: []string{"1001"},
		},
		{
			name:  "a trailing backslash never reaches the CRM",
			crm:   map[string][]*contacts.Contact{},
			req:   Request{Criteria: []model.ContactIdentificationCriterion{{Type: model.IdentificationCriterionEmail, Value: `qa@example.com\`}}},
			want:  []string{},
			terms: []string{"qa@example.com"},
		},
		{
			name: "phone criterion: tail of a longer number matches, a short tail does not",
			crm: map[string][]*contacts.Contact{"phones": {
				phoneContact("1", "+380 11 000 00 01"),
				phoneContact("2", "00001"),
			}},
			req:   Request{Criteria: []model.ContactIdentificationCriterion{phone("0110000001")}},
			want:  []string{"1:phone:unverified"},
			terms: []string{"110000001"},
		},
		{
			name:  "last N digits anchored at the end, searched by the last N",
			crm:   map[string][]*contacts.Contact{"phones": {phoneContact("1", "380000096101"), phoneContact("2", "380000961017")}},
			req:   Request{Criteria: []model.ContactIdentificationCriterion{phone("000 009 61 01")}, PhoneLastDigits: 9},
			want:  []string{"1:phone:unverified"},
			terms: []string{"000096101"},
		},
		{
			name:  "primary only skips secondary phones",
			crm:   map[string][]*contacts.Contact{"phones": {phoneContact("1", "380000000009", "380000000001")}},
			req:   Request{Criteria: []model.ContactIdentificationCriterion{phone("380000000001")}, PrimaryOnly: true},
			want:  []string{},
			terms: []string{"000000001"},
		},
		{
			name:  "primary only matches the primary phone",
			crm:   map[string][]*contacts.Contact{"phones": {phoneContact("1", "380000000001", "380000000009")}},
			req:   Request{Criteria: []model.ContactIdentificationCriterion{phone("380000000001")}, PrimaryOnly: true},
			want:  []string{"1:phone:unverified"},
			terms: []string{"000000001"},
		},
		{
			name:  "any scope matches a secondary phone",
			crm:   map[string][]*contacts.Contact{"phones": {phoneContact("1", "380000000009", "380000000001")}},
			req:   Request{Criteria: []model.ContactIdentificationCriterion{phone("380000000001")}},
			want:  []string{"1:phone:unverified"},
			terms: []string{"000000001"},
		},
		{
			name:  "a whole international number finds the number stored in its local form",
			crm:   map[string][]*contacts.Contact{"phones": {phoneContact("1", "0000000001")}},
			req:   Request{Criteria: []model.ContactIdentificationCriterion{phone("+380000000001")}},
			want:  []string{"1:phone:unverified"},
			terms: []string{"000000001"},
		},
		{
			name: "an email without @ or shorter than 5 characters never reaches the CRM",
			crm:  map[string][]*contacts.Contact{},
			req: Request{Criteria: []model.ContactIdentificationCriterion{
				{Type: model.IdentificationCriterionEmail, Value: "a"},
				{Type: model.IdentificationCriterionEmail, Value: "a@b"},
				{Type: model.IdentificationCriterionEmail, Value: "example.com"},
			}},
			want:  []string{},
			terms: []string{},
		},
		{
			name: "attribute: a key with a space is searched, bool values match",
			crm: map[string][]*contacts.Contact{"variables": {
				{Id: "1", Variables: &contacts.VariableList{Data: []*contacts.Variable{{Key: "Customer VIP", Value: structpb.NewBoolValue(true)}}}},
				{Id: "2", Variables: &contacts.VariableList{Data: []*contacts.Variable{{Key: "Customer VIP", Value: structpb.NewBoolValue(false)}}}},
			}},
			req:   Request{Criteria: []model.ContactIdentificationCriterion{{Type: model.IdentificationCriterionAttribute, Key: "Customer VIP", Value: "true"}}},
			want:  []string{"1:attribute:unverified"},
			terms: []string{"Customer VIP=true"},
		},
		{
			name: "email: case-insensitive equality, substring rejected, operators stripped",
			crm: map[string][]*contacts.Contact{"emails": {
				{Id: "1", Emails: &contacts.EmailList{Data: []*contacts.EmailAddress{{Email: "QA@example.com"}}}},
				{Id: "2", Emails: &contacts.EmailList{Data: []*contacts.EmailAddress{{Email: "xqa@example.com"}}}},
			}},
			req:   Request{Criteria: []model.ContactIdentificationCriterion{{Type: model.IdentificationCriterionEmail, Value: "qa@example.com"}, {Type: model.IdentificationCriterionEmail, Value: "/=qa*@example.com/"}}},
			want:  []string{"1:email:unverified"},
			terms: []string{"qa@example.com", "qa@example.com"},
		},
		{
			name: "attribute: number values match, other keys do not",
			crm: map[string][]*contacts.Contact{"variables": {
				{Id: "1", Variables: &contacts.VariableList{Data: []*contacts.Variable{{Key: "contract", Value: structpb.NewNumberValue(1001)}}}},
				{Id: "2", Variables: &contacts.VariableList{Data: []*contacts.Variable{{Key: "other", Value: structpb.NewStringValue("1001")}}}},
			}},
			req:   Request{Criteria: []model.ContactIdentificationCriterion{{Type: model.IdentificationCriterionAttribute, Key: "contract", Value: "1001"}}},
			want:  []string{"1:attribute:unverified"},
			terms: []string{"contract=1001"},
		},
		{
			name: "unsafe or empty criteria never reach the CRM",
			crm:  map[string][]*contacts.Contact{},
			req: Request{Criteria: []model.ContactIdentificationCriterion{
				phone("' OR 1=1 --"),
				{Type: model.IdentificationCriterionAttribute, Key: "a=b", Value: "x"},
				{Type: model.IdentificationCriterionAttribute, Key: "a*", Value: "x"},
				{Type: model.IdentificationCriterionAttribute, Key: "/a/", Value: "x"},
				{Type: model.IdentificationCriterionAttribute, Key: strings.Repeat("k", maxKeyLen+1), Value: "x"},
				{Type: model.IdentificationCriterionAttribute, Key: "k", Value: "%%"},
			}},
			want:  []string{},
			terms: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			crm := &fakeCRM{byQin: tt.crm}
			tt.req.DomainID = 1

			got, _, err := Search(context.Background(), crm, tt.req)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if !slices.Equal(ids(got), tt.want) {
				t.Fatalf("matches = %v, want %v", ids(got), tt.want)
			}

			terms := []string{}
			for _, r := range crm.reqs {
				terms = append(terms, r.GetQ())

				if r.GetDomainId() != 1 {
					t.Fatalf("domain = %d, want 1", r.GetDomainId())
				}
			}

			if !slices.Equal(terms, tt.terms) {
				t.Fatalf("CRM terms = %v, want %v", terms, tt.terms)
			}
		})
	}
}

func TestSearch_CallerNumberIsOnePage(t *testing.T) {
	t.Parallel()

	rows := make([]*contacts.Contact, 0, 50)
	for i := range 50 {
		rows = append(rows, phoneContact(strconv.Itoa(i), "380000000001"))
	}

	crm := &fakeCRM{byQin: map[string][]*contacts.Contact{"phones": rows}}

	got, _, err := Search(context.Background(), crm, Request{DomainID: 1, CallerNumber: "380000000001"})
	if err != nil {
		t.Fatal(err)
	}

	if len(crm.reqs) != 1 || len(got) != callerPageSize {
		t.Fatalf("requests = %d, matches = %d; want 1 and %d", len(crm.reqs), len(got), callerPageSize)
	}
}

func TestSearch_PagesCriteria(t *testing.T) {
	t.Parallel()

	rows := make([]*contacts.Contact, 0, 250)
	for i := range 250 {
		rows = append(rows, phoneContact(strconv.Itoa(i), "380000000500"))
	}

	rows[230] = phoneContact("hit", "380000000001")
	crm := &fakeCRM{byQin: map[string][]*contacts.Contact{"phones": rows}}

	got, _, err := Search(context.Background(), crm, Request{DomainID: 1, Criteria: []model.ContactIdentificationCriterion{phone("380000000001")}})
	if err != nil {
		t.Fatal(err)
	}

	if !slices.Equal(ids(got), []string{"hit:phone:unverified"}) || len(crm.reqs) != 3 {
		t.Fatalf("matches = %v after %d requests; want the hit on page 3", ids(got), len(crm.reqs))
	}
}

func TestSearch_CRMErrorIsReturned(t *testing.T) {
	t.Parallel()

	crm := &fakeCRM{err: model.NewAppError("test", "test", nil, "down", 503)}
	if _, _, err := Search(context.Background(), crm, Request{DomainID: 1, CallerNumber: "380000000001"}); err == nil {
		t.Fatal("expected the CRM error")
	}
}

func TestMergeAndSummary(t *testing.T) {
	t.Parallel()

	callerA := Match{ContactID: "A", Reason: Reason{Type: typeCallerNumber, Trust: model.IdentificationTrustVerified, Origin: originChannel}}
	flowB := Match{ContactID: "B", Reason: Reason{Type: "phone", Trust: model.IdentificationTrustUnverified, Origin: originFlow}}
	flowA := Match{ContactID: "A", Reason: Reason{Type: "email", Trust: model.IdentificationTrustUnverified, Origin: originFlow}}

	all, changed := Merge(nil, []Match{flowB}, 10)
	if !changed || len(all) != 1 {
		t.Fatalf("first merge: changed=%v len=%d", changed, len(all))
	}

	if s := Summary(all, "B"); s.FoundCount != 1 || s.ContactID != "" || s.Trust != model.IdentificationTrustUnverified {
		t.Fatalf("single unverified summary = %+v; contact_id must stay empty", s)
	}

	if _, ok := SingleVerified(all); ok {
		t.Fatal("an unverified candidate must never be bound")
	}

	all, changed = Merge(all, []Match{callerA, flowA}, 20)
	if !changed || len(all) != 2 || all[0].ContactID != "A" || len(all[0].Reasons) != 2 {
		t.Fatalf("second merge = %+v, want A first with 2 reasons", all)
	}

	if id, ok := SingleVerified(all); !ok || id != "A" {
		t.Fatalf("single verified = %q %v, want A", id, ok)
	}

	if s := Summary(all, "A"); s.FoundCount != 2 || s.ContactID != "" || s.Trust != model.IdentificationTrustVerified {
		t.Fatalf("summary = %+v; contact_id only for exactly one candidate", s)
	}

	again, changed := Merge(all, []Match{callerA, flowA, flowB}, 30)
	if changed || !slices.EqualFunc(again, all, func(a, b Candidate) bool { return a.ContactID == b.ContactID && len(a.Reasons) == len(b.Reasons) }) {
		t.Fatal("the same matches twice must not change anything")
	}

	two, _ := Merge(all, []Match{{ContactID: "C", Reason: callerA.Reason}}, 40)
	if _, ok := SingleVerified(two); ok {
		t.Fatal("two verified candidates must not be bound automatically")
	}
}

func TestMerge_Cap(t *testing.T) {
	t.Parallel()

	matches := make([]Match, 0, maxCandidates+5)
	for i := range maxCandidates + 5 {
		matches = append(matches, Match{ContactID: strconv.Itoa(i), Reason: Reason{Type: "phone", Trust: model.IdentificationTrustUnverified, Origin: originFlow}})
	}

	all, _ := Merge(nil, matches, 1)
	if len(all) != maxCandidates {
		t.Fatalf("len = %d, want %d", len(all), maxCandidates)
	}
}

func TestPhoneMatches(t *testing.T) {
	t.Parallel()

	tests := []struct {
		stored, wanted string
		lastN          int
		want           bool
	}{
		{"+380 (11) 000-00-01", "380110000001", 0, true},
		{"0110000001", "380110000001", 0, true},
		{"110000001", "380110000001", 0, true},
		{"0001", "380110000001", 0, false},
		{"380110000002", "380110000001", 0, false},
		{"380000096101", "0000096101", 9, true},
		{"380000961017", "0000096101", 9, false},
		{"12", "0012", 4, false},
		{"", "380110000001", 0, false},
		{"1001", "1001", 0, true},
	}

	for _, tt := range tests {
		if got := phoneMatches(tt.stored, tt.wanted, tt.lastN); got != tt.want {
			t.Errorf("phoneMatches(%q, %q, %d) = %v, want %v", tt.stored, tt.wanted, tt.lastN, got, tt.want)
		}
	}
}

func TestArgsCriteria(t *testing.T) {
	t.Parallel()

	args := model.ContactIdentificationArgs{Variables: []model.ContactIdentificationVariable{
		{Key: "phone", Value: "+380000000001"},
		{Key: "Email", Value: "qa@example.com"},
		{Key: "contract", Value: "A-1"},
		{Key: "phone", Value: "  "},
		{Key: "", Value: "x"},
	}, PhoneScope: "Primary"}

	got := args.Criteria()
	if len(got) != 3 || got[0].Type != model.IdentificationCriterionPhone || got[1].Type != model.IdentificationCriterionEmail ||
		got[2].Type != model.IdentificationCriterionAttribute || got[2].Key != "contract" {
		t.Fatalf("criteria = %+v", got)
	}

	if !args.PrimaryPhonesOnly() {
		t.Fatal("phoneScope primary must be primary-only")
	}
}

func TestSummary_ContactIDOnlyForTheContactOnTheCall(t *testing.T) {
	t.Parallel()

	verified := []Candidate{{ContactID: "A", Trust: model.IdentificationTrustVerified}}

	tests := []struct {
		name   string
		all    []Candidate
		onCall string
		want   string
	}{
		{name: "bound now", all: verified, onCall: "A", want: "A"},
		{name: "the call already carries another contact", all: verified, onCall: "X", want: ""},
		{name: "bind failed", all: verified, onCall: "", want: ""},
		{name: "unverified on the call", all: []Candidate{{ContactID: "A", Trust: model.IdentificationTrustUnverified}}, onCall: "A", want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := Summary(tt.all, tt.onCall); got.ContactID != tt.want || got.FoundCount != len(tt.all) {
				t.Fatalf("Summary = %+v, want contact_id %q", got, tt.want)
			}
		})
	}
}

func TestDuplicates(t *testing.T) {
	t.Parallel()

	v := Reason{Type: typeCallerNumber, Trust: model.IdentificationTrustVerified, Origin: originChannel}
	u := Reason{Type: "email", Trust: model.IdentificationTrustUnverified, Origin: originFlow}

	if d := Duplicates([]Candidate{{ContactID: "B", Trust: model.IdentificationTrustVerified, Reasons: []Reason{v}}, {ContactID: "C", Trust: model.IdentificationTrustUnverified, Reasons: []Reason{u}}}); d != nil {
		t.Fatalf("one verified candidate is no duplicate: %+v", d)
	}

	all := []Candidate{
		{ContactID: "B", Trust: model.IdentificationTrustVerified, Reasons: []Reason{v, u}},
		{ContactID: "A", Trust: model.IdentificationTrustVerified, Reasons: []Reason{v}},
		{ContactID: "C", Trust: model.IdentificationTrustUnverified, Reasons: []Reason{u}},
	}

	d := Duplicates(all)
	if d == nil || !slices.Equal(d.ContactIDs, []string{"A", "B"}) {
		t.Fatalf("duplicate = %+v, want sorted A,B", d)
	}

	if len(d.Reasons["B"]) != 1 || d.Reasons["B"][0] != v {
		t.Fatalf("reasons of B = %+v, want only the verified one", d.Reasons["B"])
	}

	again := Duplicates(all)
	if !slices.Equal(again.ContactIDs, d.ContactIDs) {
		t.Fatal("the same candidates must give the same record")
	}
}

func TestCallerMatches(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name, stored, caller string
		want                 bool
	}{
		{"same number", "+380 11 000 00 01", "380110000001", true},
		{"local form, UA", "0110000001", "380110000001", true},
		{"trunk prefix 0, UK", "020 7946 0958", "442079460958", true},
		{"trunk prefix 0, DE", "030 12345678", "493012345678", true},
		{"stored without country, 7 digits", "0000001", "380110000001", true},
		{"other number, same last 8", "380120000001", "380110000001", false},
		{"too short", "00001", "380110000001", false},
		{"caller digits only in the middle", "380110000001999", "380110000001", false},
		{"empty stored", "", "380110000001", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := callerMatches(tt.stored, tt.caller); got != tt.want {
				t.Fatalf("callerMatches(%q, %q) = %v, want %v", tt.stored, tt.caller, got, tt.want)
			}
		})
	}
}

func TestCallerTrust(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name, stored, caller, want string
	}{
		{"same number", "+380 11 000 00 01", "380110000001", model.IdentificationTrustVerified},
		{"local form, UA", "0110000001", "380110000001", model.IdentificationTrustVerified},
		{"trunk prefix 0, UK", "020 7946 0958", "442079460958", model.IdentificationTrustVerified},
		{"trunk prefix 0, DE", "030 12345678", "493012345678", model.IdentificationTrustVerified},
		{"caller dialed locally, stored international", "+380110000001", "0110000001", model.IdentificationTrustVerified},
		{"another country, same last 9 digits", "+48 110000001", "380110000001", model.IdentificationTrustUnverified},
		{"stored without country code or trunk prefix", "110000001", "380110000001", model.IdentificationTrustVerified},
		{"caller without country code (trunk sends 9 digits)", "+380110000001", "110000001", model.IdentificationTrustVerified},
		{"international prefix 00", "+380110000001", "00380110000001", model.IdentificationTrustVerified},
		{"8 digits without country code is only a tail", "10000001", "380110000001", model.IdentificationTrustUnverified},
		{"short local number, same last 7 digits", "0000001", "380110000001", model.IdentificationTrustUnverified},
		{"other number", "380120000001", "380110000001", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := callerTrust(tt.stored, tt.caller); got != tt.want {
				t.Fatalf("callerTrust(%q, %q) = %q, want %q", tt.stored, tt.caller, got, tt.want)
			}
		})
	}
}

func TestSearch_CallerNumberTrunkPrefix(t *testing.T) {
	t.Parallel()

	crm := &fakeCRM{byQin: map[string][]*contacts.Contact{
		"phones": {phoneContact("10", "020 7946 0958"), phoneContact("11", "+44 20 7946 0959")},
	}}

	matches, _, err := Search(context.Background(), crm, Request{DomainID: 1, CallerNumber: "+44 20 7946 0958"})
	if err != nil {
		t.Fatal(err)
	}

	if len(matches) != 1 || matches[0].ContactID != "10" || matches[0].Reason.Trust != model.IdentificationTrustVerified {
		t.Fatalf("matches = %+v, want only 10, verified", matches)
	}
}

// A number of another country that only shares the caller's last 9 digits
// is a possible match, never bound to the call.
func TestSearch_CallerTailOnlyIsUnverified(t *testing.T) {
	t.Parallel()

	crm := &fakeCRM{byQin: map[string][]*contacts.Contact{
		"phones": {phoneContact("20", "+48 110000001"), phoneContact("21", "+380 11 000 00 01")},
	}}

	matches, _, err := Search(context.Background(), crm, Request{DomainID: 1, CallerNumber: "+380110000001"})
	if err != nil {
		t.Fatal(err)
	}

	trust := map[string]string{}
	for _, m := range matches {
		trust[m.ContactID] = m.Reason.Trust
	}

	if trust["20"] != model.IdentificationTrustUnverified || trust["21"] != model.IdentificationTrustVerified {
		t.Fatalf("trust = %v, want 20 unverified, 21 verified", trust)
	}

	all, _ := Merge(nil, matches, 1)
	if id, ok := SingleVerified(all); !ok || id != "21" {
		t.Fatalf("bound %q/%v, want 21", id, ok)
	}
}

func TestArgsCriteria_Limits(t *testing.T) {
	t.Parallel()

	vars := make([]model.ContactIdentificationVariable, 0, 20)
	vars = append(vars,
		model.ContactIdentificationVariable{Key: "phone", Value: strings.Repeat("1", model.IdentificationMaxValueLen+1)},
		model.ContactIdentificationVariable{Key: strings.Repeat("k", model.IdentificationMaxKeyLen+1), Value: "v"},
		model.ContactIdentificationVariable{Key: "email", Value: strings.Repeat("я", model.IdentificationMaxValueLen)},
	)

	for i := range 20 {
		vars = append(vars, model.ContactIdentificationVariable{Key: "attr" + strconv.Itoa(i), Value: "v"})
	}

	got := model.ContactIdentificationArgs{Variables: vars}.Criteria()
	if len(got) != model.IdentificationMaxCriteria {
		t.Fatalf("len = %d, want %d", len(got), model.IdentificationMaxCriteria)
	}

	if got[0].Type != model.IdentificationCriterionEmail {
		t.Fatalf("first = %+v; a value of 256 characters (not bytes) is kept, longer ones and long keys dropped", got[0])
	}
}

func TestArgsPhoneLastDigits(t *testing.T) {
	t.Parallel()

	tests := []struct {
		in, want int
		ok       bool
	}{
		{0, 0, true}, {4, 4, true}, {15, 15, true}, {3, 0, false}, {1, 0, false}, {16, 0, false}, {-2, 0, false},
	}

	for _, tt := range tests {
		got, ok := model.ContactIdentificationArgs{PhoneLastDigits: tt.in}.PhoneLastDigitsOrWhole()
		if got != tt.want || ok != tt.ok {
			t.Errorf("PhoneLastDigitsOrWhole(%d) = %d,%v want %d,%v", tt.in, got, ok, tt.want, tt.ok)
		}
	}
}

func TestSearch_FailedCriterionIsSkipped(t *testing.T) {
	t.Parallel()

	down := model.NewAppError("test", "test", nil, "down", 503)
	email := model.ContactIdentificationCriterion{Type: model.IdentificationCriterionEmail, Value: "qa@example.com"}
	attr := model.ContactIdentificationCriterion{Type: model.IdentificationCriterionAttribute, Key: "contract", Value: "A-1"}
	rows := map[string][]*contacts.Contact{
		"phones":    {phoneContact("1", "+380000000001")},
		"variables": {{Id: "2", Variables: &contacts.VariableList{Data: []*contacts.Variable{{Key: "contract", Value: structpb.NewStringValue("A-1")}}}}},
	}

	tests := []struct {
		name     string
		failing  []string
		criteria []model.ContactIdentificationCriterion
		want     []string
		skipped  int
		wantErr  bool
	}{
		{
			name:     "one criterion fails, the other and the caller are kept",
			failing:  []string{"emails"},
			criteria: []model.ContactIdentificationCriterion{email, attr},
			want:     []string{"1:caller_number:verified", "2:attribute:unverified"},
			skipped:  1,
		},
		{
			name:     "every criterion fails: the run fails but keeps the caller matches",
			failing:  []string{"emails", "variables"},
			criteria: []model.ContactIdentificationCriterion{email, attr},
			want:     []string{"1:caller_number:verified"},
			skipped:  2,
			wantErr:  true,
		},
		{
			name:     "an unusable criterion is not a failure",
			failing:  []string{"emails"},
			criteria: []model.ContactIdentificationCriterion{email, attr, phone("12")},
			want:     []string{"1:caller_number:verified", "2:attribute:unverified"},
			skipped:  1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			crm := &fakeCRM{byQin: rows, errByQin: map[string]*model.AppError{}}
			for _, q := range tt.failing {
				crm.errByQin[q] = down
			}

			got, skipped, err := Search(context.Background(), crm, Request{DomainID: 1, CallerNumber: "+380000000001", Criteria: tt.criteria})
			if (err != nil) != tt.wantErr || skipped != tt.skipped {
				t.Fatalf("err = %v, skipped = %d; want error %v, skipped %d", err, skipped, tt.wantErr, tt.skipped)
			}

			if !slices.Equal(ids(got), tt.want) {
				t.Fatalf("matches = %v, want %v", ids(got), tt.want)
			}
		})
	}
}

func TestSearch_NestedListsAreNotCut(t *testing.T) {
	t.Parallel()

	crm := &fakeCRM{byQin: map[string][]*contacts.Contact{}}

	_, _, err := Search(context.Background(), crm, Request{DomainID: 1, CallerNumber: "+380000000001", Criteria: []model.ContactIdentificationCriterion{
		phone("+380000000002"),
		{Type: model.IdentificationCriterionEmail, Value: "qa@example.com"},
		{Type: model.IdentificationCriterionAttribute, Key: "contract", Value: "A-1"},
	}})
	if err != nil {
		t.Fatal(err)
	}

	want := [][]string{fieldsCaller, fieldsPhones, fieldsEmails, fieldsVariables}
	if len(crm.reqs) != len(want) {
		t.Fatalf("requests = %d, want %d", len(crm.reqs), len(want))
	}

	for i, r := range crm.reqs {
		if !slices.Equal(r.GetFields(), want[i]) || !strings.Contains(r.GetFields()[1], ".size(64)") {
			t.Fatalf("request %d fields = %v, want %v", i, r.GetFields(), want[i])
		}
	}
}

func TestMerge_CapKeepsVerified(t *testing.T) {
	t.Parallel()

	unverified := make([]Match, 0, maxCandidates)
	for i := range maxCandidates {
		unverified = append(unverified, Match{ContactID: "u" + strconv.Itoa(i), Reason: Reason{Type: "phone", Trust: model.IdentificationTrustUnverified, Origin: originFlow}})
	}

	all, _ := Merge(nil, unverified, 1)

	verified := make([]Match, 0, 2*maxCandidates)
	for i := range 2 * maxCandidates {
		verified = append(verified, Match{ContactID: "v" + strconv.Itoa(i), Reason: Reason{Type: typeCallerNumber, Trust: model.IdentificationTrustVerified, Origin: originChannel}})
	}

	all, _ = Merge(all, append([]Match{{ContactID: "late", Reason: unverified[0].Reason}}, verified...), 2)
	if len(all) != 2*maxCandidates {
		t.Fatalf("len = %d, want %d", len(all), 2*maxCandidates)
	}

	if all[0].Trust != model.IdentificationTrustVerified || slices.ContainsFunc(all, func(c Candidate) bool { return c.ContactID == "late" }) {
		t.Fatal("past the bound only verified contacts are added, and they come first")
	}
}

func TestMerge_SameReasonGainsTrust(t *testing.T) {
	t.Parallel()

	tail := Match{ContactID: "A", Reason: Reason{Type: typeCallerNumber, Trust: model.IdentificationTrustUnverified, Origin: originChannel}}
	exact := Match{ContactID: "A", Reason: Reason{Type: typeCallerNumber, Trust: model.IdentificationTrustVerified, Origin: originChannel}}

	all, _ := Merge(nil, []Match{tail}, 1)

	all, changed := Merge(all, []Match{exact}, 2)
	if !changed || len(all) != 1 || len(all[0].Reasons) != 1 || all[0].Trust != model.IdentificationTrustVerified ||
		all[0].Reasons[0].Trust != model.IdentificationTrustVerified {
		t.Fatalf("merge = %+v, want one reason upgraded to verified", all)
	}

	if _, changed := Merge(all, []Match{tail}, 3); changed {
		t.Fatal("a weaker equal reason must not downgrade or change anything")
	}
}

func TestMerge_OrderTieBreakByID(t *testing.T) {
	t.Parallel()

	r := Reason{Type: "phone", Trust: model.IdentificationTrustUnverified, Origin: originFlow}

	all, _ := Merge(nil, []Match{{ContactID: "C", Reason: r}, {ContactID: "A", Reason: r}, {ContactID: "B", Reason: r}}, 1)
	if got := []string{all[0].ContactID, all[1].ContactID, all[2].ContactID}; !slices.Equal(got, []string{"A", "B", "C"}) {
		t.Fatalf("order = %v, want by id when trust and time are equal", got)
	}
}

func TestPhoneMatches_SuffixBoundary(t *testing.T) {
	t.Parallel()

	if !phoneMatches("1000001", "380001000001", 0) {
		t.Fatal("a 7-digit tail must match")
	}

	if phoneMatches("000001", "380001000001", 0) {
		t.Fatal("a 6-digit tail must not match")
	}
}

func TestArgsPrimaryPhonesOnly(t *testing.T) {
	t.Parallel()

	for scope, want := range map[string]bool{
		"": false, "any": false, "Any": false, "primary": true, "Primary": true, "primary_only": true, "Primary only": true, "PRIMARY-ONLY": true, "secondary": false,
	} {
		if got := (model.ContactIdentificationArgs{PhoneScope: scope}).PrimaryPhonesOnly(); got != want {
			t.Errorf("PrimaryPhonesOnly(%q) = %v, want %v", scope, got, want)
		}
	}
}

// Keys are bounded in characters, like the API and the node's arguments.
func TestIsSearchableKey_LengthInCharacters(t *testing.T) {
	t.Parallel()

	if !isSearchableKey(strings.Repeat("я", maxKeyLen)) {
		t.Fatal("a key of maxKeyLen letters is searchable")
	}

	if isSearchableKey(strings.Repeat("я", maxKeyLen+1)) {
		t.Fatal("a key over maxKeyLen letters is not searchable")
	}
}
