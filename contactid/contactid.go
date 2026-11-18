// Package contactid identifies the CRM contact of a call: it searches the CRM
// by the caller's number (verified) and by schema criteria (unverified), keeps
// the candidates and tells whether exactly one verified contact can be bound.
// Chats do the same in im-thread-service, and the matching rules here are kept
// identical to it, so the node behaves the same in both schema types.
package contactid

import (
	"cmp"
	"context"
	"slices"
	"strconv"
	"strings"
	"unicode/utf8"

	"google.golang.org/protobuf/types/known/structpb"

	"github.com/webitel/flow_manager/gen/contacts"
	"github.com/webitel/flow_manager/model"
)

const (
	// pageSize and maxPages bound one CRM search. The CRM matches substrings,
	// so rows are re-checked here: a broader CRM match costs rows, never a
	// wrong candidate.
	pageSize = 100
	maxPages = 5
	// callerPageSize bounds the caller-number search, which runs on every call
	// with auto-link on: more than one match binds nothing anyway.
	callerPageSize = 20
	// maxCandidates bounds the candidate list of one call: new unverified
	// contacts past it are dropped, known ones still gain reasons. Verified
	// contacts are still added, up to twice the bound, so a list full of
	// possible matches cannot hide the one that may be bound.
	maxCandidates = 100
	// minPhoneDigits keeps a phone criterion from matching most of the CRM.
	minPhoneDigits = 4
	// minEmailLen keeps a one-letter "email" from scanning the CRM.
	minEmailLen = 5
	// maxKeyLen bounds an attribute key, like the node arguments do.
	maxKeyLen = model.IdentificationMaxKeyLen
	// minSuffixDigits is the shortest number accepted as the tail of a longer
	// one (+380000000001 vs 0000000001) when no last-N is configured.
	minSuffixDigits = 7
	// nationalDigits is how many trailing digits identify a number: the
	// national significant number, which a trunk prefix (0) or a country code
	// in front does not change. It is also the CRM search term when no last-N
	// is asked: the CRM finds a number only if the term is inside it, so the
	// whole international number would miss the same number stored in its
	// local form. The whole number is still what rows are compared with.
	nationalDigits = 9
	// minLocalNationalDigits is the shortest national number that, written
	// with a trunk prefix (0…), is taken for the same number as the caller's.
	minLocalNationalDigits = 8
	// minIntlDigits is the shortest number taken as international after a 00.
	minIntlDigits = 10

	originChannel = "channel"
	originFlow    = "flow"
	// typeCallerNumber is the reason of a match by the caller's number.
	typeCallerNumber = "caller_number"
)

// CRM lists nested in a contact are cut to 16 rows by default; 64 is the most
// the CRM returns.
var (
	fieldsCaller    = []string{"id", "phones.size(64){number}"}
	fieldsPhones    = []string{"id", "phones.size(64){number,primary}"}
	fieldsEmails    = []string{"id", "emails.size(64){email}"}
	fieldsVariables = []string{"id", "variables.size(64){key,value}"}
)

// Reason is why a contact became a candidate; never the matched value.
type Reason struct {
	Type   string `json:"type"`
	Trust  string `json:"trust"`
	Origin string `json:"origin"`
	Key    string `json:"key,omitempty"`
}

// Candidate is a CRM contact identified for a call.
type Candidate struct {
	ContactID string   `json:"contact_id"`
	Trust     string   `json:"trust"`
	Reasons   []Reason `json:"reasons"`
	CreatedAt int64    `json:"created_at"`
}

// Match is one contact found by one search.
type Match struct {
	ContactID string
	Reason    Reason
}

// Searcher is the CRM search the package needs (flow manager's SearchContactsNA).
type Searcher interface {
	SearchContactsNA(ctx context.Context, req *contacts.SearchContactsNARequest) (*contacts.ContactList, *model.AppError)
}

// Request is one identification run.
type Request struct {
	DomainID int64
	// CallerNumber is verified: a stored number with the same national tail
	// (see callerTrust).
	CallerNumber string
	// Criteria are already bounded by the node arguments (Criteria()).
	Criteria        []model.ContactIdentificationCriterion
	PhoneLastDigits int
	PrimaryOnly     bool
}

func trustRank(t string) int {
	switch t {
	case model.IdentificationTrustVerified:
		return 2
	case model.IdentificationTrustUnverified:
		return 1
	default:
		return 0
	}
}

func maxTrust(a, b string) string {
	if trustRank(b) > trustRank(a) {
		return b
	}

	return a
}

// Search runs the caller-number search and every criterion (OR). A failed
// caller-number search fails the run. A criterion whose search fails is
// skipped and counted, so the others still run; only when every criterion
// failed is err set, and then the caller-number matches are still returned:
// the caller's own number may be kept and bound even while the CRM refuses the
// schema's criteria.
func Search(ctx context.Context, s Searcher, req Request) (matches []Match, skipped int, err *model.AppError) {
	if digits := phoneDigits(req.CallerNumber); len(digits) >= minPhoneDigits {
		found, err := searchPages(ctx, s, req.DomainID, "phones", phoneTerm(digits, 0), fieldsCaller, callerPageSize, 1)
		if err != nil {
			return nil, 0, err
		}

		// The CRM matches an unanchored substring; keep only real matches. Only
		// the caller's very number is verified and may be bound; a stored number
		// sharing just its tail may belong to someone else.
		for _, c := range found {
			trust := ""

			for _, p := range c.GetPhones().GetData() {
				if t := callerTrust(p.GetNumber(), digits); trustRank(t) > trustRank(trust) {
					trust = t
				}
			}

			if trust == "" {
				continue
			}

			matches = append(matches, Match{ContactID: c.GetId(), Reason: Reason{
				Type: typeCallerNumber, Trust: trust, Origin: originChannel,
			}})
		}
	}

	var lastErr *model.AppError

	for _, cr := range req.Criteria {
		ids, searched, err := searchCriterion(ctx, s, req, cr)
		if err != nil {
			skipped++
			lastErr = err

			continue
		}

		if !searched {
			continue
		}

		reason := Reason{Type: cr.Type, Trust: model.IdentificationTrustUnverified, Origin: originFlow}
		if cr.Type == model.IdentificationCriterionAttribute {
			reason.Key = cr.Key
		}

		for _, id := range ids {
			matches = append(matches, Match{ContactID: id, Reason: reason})
		}
	}

	if skipped > 0 && skipped == searchable(req.Criteria) {
		// The CRM is down for the criteria: "found nothing" would mislead the
		// schema.
		return matches, skipped, lastErr
	}

	return matches, skipped, nil
}

// searchable counts the criteria that are sent to the CRM.
func searchable(criteria []model.ContactIdentificationCriterion) int {
	n := 0

	for _, cr := range criteria {
		if usable(cr) {
			n++
		}
	}

	return n
}

// usable reports whether a criterion can be searched: what a client typed
// instead of a phone, say, is skipped without a CRM call.
func usable(cr model.ContactIdentificationCriterion) bool {
	switch cr.Type {
	case model.IdentificationCriterionPhone:
		return len(phoneDigits(cr.Value)) >= minPhoneDigits
	case model.IdentificationCriterionEmail:
		t := substringTerm(cr.Value)

		return len(t) >= minEmailLen && strings.Contains(t, "@")
	case model.IdentificationCriterionAttribute:
		return isSearchableKey(cr.Key) && substringTerm(cr.Value) != ""
	default:
		return false
	}
}

// searchCriterion searches one criterion; searched is false for a criterion
// that cannot be searched.
func searchCriterion(ctx context.Context, s Searcher, req Request, cr model.ContactIdentificationCriterion) (ids []string, searched bool, err *model.AppError) {
	if !usable(cr) {
		return nil, false, nil
	}

	switch cr.Type {
	case model.IdentificationCriterionPhone:
		digits := phoneDigits(cr.Value)

		found, err := searchAll(ctx, s, req.DomainID, "phones", phoneTerm(digits, req.PhoneLastDigits), fieldsPhones)
		if err != nil {
			return nil, true, err
		}

		for _, c := range found {
			for _, p := range c.GetPhones().GetData() {
				if req.PrimaryOnly && !p.GetPrimary() {
					continue
				}

				if phoneMatches(p.GetNumber(), digits, req.PhoneLastDigits) {
					ids = append(ids, c.GetId())

					break
				}
			}
		}
	case model.IdentificationCriterionEmail:
		found, err := searchAll(ctx, s, req.DomainID, "emails", substringTerm(cr.Value), fieldsEmails)
		if err != nil {
			return nil, true, err
		}

		for _, c := range found {
			if slices.ContainsFunc(c.GetEmails().GetData(), func(e *contacts.EmailAddress) bool { return emailMatches(e.GetEmail(), cr.Value) }) {
				ids = append(ids, c.GetId())
			}
		}
	case model.IdentificationCriterionAttribute:
		found, err := searchAll(ctx, s, req.DomainID, "variables", cr.Key+"="+substringTerm(cr.Value), fieldsVariables)
		if err != nil {
			return nil, true, err
		}

		want := strings.TrimSpace(cr.Value)

		for _, c := range found {
			if slices.ContainsFunc(c.GetVariables().GetData(), func(v *contacts.Variable) bool {
				return v.GetKey() == cr.Key && scalarString(v.GetValue()) == want
			}) {
				ids = append(ids, c.GetId())
			}
		}
	}

	return ids, true, nil
}

// phoneTerm is what the CRM is asked for: the last N digits when asked, else
// the national tail of the number.
func phoneTerm(digits string, lastN int) string {
	n := nationalDigits
	if lastN > 0 {
		n = lastN
	}

	if len(digits) > n {
		return digits[len(digits)-n:]
	}

	return digits
}

func searchAll(ctx context.Context, s Searcher, domainID int64, qin, q string, fields []string) ([]*contacts.Contact, *model.AppError) {
	return searchPages(ctx, s, domainID, qin, q, fields, pageSize, maxPages)
}

func searchPages(ctx context.Context, s Searcher, domainID int64, qin, q string, fields []string, size, pages int32) ([]*contacts.Contact, *model.AppError) {
	if q == "" {
		// An empty term would match every contact of the domain.
		return nil, nil
	}

	var out []*contacts.Contact

	for page := int32(1); page <= pages; page++ {
		list, err := s.SearchContactsNA(ctx, &contacts.SearchContactsNARequest{
			DomainId: domainID,
			Qin:      []string{qin},
			Q:        q,
			Fields:   fields,
			Size:     size,
			Page:     page,
		})
		if err != nil {
			return nil, err
		}

		out = append(out, list.GetData()...)

		if !list.GetNext() {
			break
		}
	}

	return out, nil
}

// Merge appends matches to the existing candidates, never replacing them:
// new contacts are added, known ones gain the new reasons, trust is the max.
// It returns the list (verified first) and whether anything changed.
func Merge(existing []Candidate, matches []Match, now int64) ([]Candidate, bool) {
	all := slices.Clone(existing)
	changed := false

	index := make(map[string]int, len(all))
	for i, c := range all {
		all[i].Reasons = slices.Clone(c.Reasons)
		index[c.ContactID] = i
	}

	for _, m := range matches {
		id := strings.TrimSpace(m.ContactID)
		if id == "" {
			continue
		}

		i, ok := index[id]
		if !ok {
			if len(all) >= 2*maxCandidates || (len(all) >= maxCandidates && m.Reason.Trust != model.IdentificationTrustVerified) {
				continue
			}

			all = append(all, Candidate{ContactID: id, CreatedAt: now})
			i = len(all) - 1
			index[id] = i
			changed = true
		}

		if addReason(&all[i], m.Reason) {
			changed = true
		}
	}

	slices.SortStableFunc(all, func(a, b Candidate) int {
		return cmp.Or(
			cmp.Compare(trustRank(b.Trust), trustRank(a.Trust)),
			cmp.Compare(a.CreatedAt, b.CreatedAt),
			cmp.Compare(a.ContactID, b.ContactID),
		)
	})

	return all, changed
}

func addReason(c *Candidate, r Reason) bool {
	for i, have := range c.Reasons {
		if have.Type != r.Type || have.Origin != r.Origin || have.Key != r.Key {
			continue
		}

		if trustRank(r.Trust) <= trustRank(have.Trust) {
			return false
		}

		c.Reasons[i].Trust = r.Trust
		c.Trust = maxTrust(c.Trust, r.Trust)

		return true
	}

	c.Reasons = append(c.Reasons, r)
	c.Trust = maxTrust(c.Trust, r.Trust)

	return true
}

// Summary is what the schema gets. contact_id is set only when there is
// exactly one candidate, it is verified and it is onCall, the contact the call
// carries after this run: a contact chosen earlier is never reported as found.
func Summary(all []Candidate, onCall string) model.ContactIdentificationResult {
	res := model.ContactIdentificationResult{FoundCount: len(all)}

	for _, c := range all {
		res.Trust = maxTrust(res.Trust, c.Trust)
	}

	if len(all) == 1 && all[0].Trust == model.IdentificationTrustVerified && all[0].ContactID == onCall {
		res.ContactID = all[0].ContactID
	}

	return res
}

// Duplicate records two or more verified candidates of one call.
// Contacts are never merged automatically.
type Duplicate struct {
	// ContactIDs are sorted, so the same set is the same record.
	ContactIDs []string `json:"contact_ids"`
	// Reasons are the verified reasons per contact.
	Reasons map[string][]Reason `json:"reasons"`
}

// Duplicates returns the call's verified candidates as one duplicate record,
// or nil when fewer than two are verified. Candidates only gain reasons and
// trust, so the latest record covers every earlier one.
func Duplicates(all []Candidate) *Duplicate {
	var d Duplicate

	for _, c := range all {
		if c.Trust != model.IdentificationTrustVerified {
			continue
		}

		d.ContactIDs = append(d.ContactIDs, c.ContactID)

		if d.Reasons == nil {
			d.Reasons = make(map[string][]Reason)
		}

		for _, r := range c.Reasons {
			if r.Trust == model.IdentificationTrustVerified {
				d.Reasons[c.ContactID] = append(d.Reasons[c.ContactID], r)
			}
		}
	}

	if len(d.ContactIDs) < 2 {
		return nil
	}

	slices.Sort(d.ContactIDs)

	return &d
}

// SingleVerified returns the only verified candidate, if there is exactly one.
func SingleVerified(all []Candidate) (string, bool) {
	var id string

	for _, c := range all {
		if c.Trust != model.IdentificationTrustVerified {
			continue
		}

		if id != "" {
			return "", false
		}

		id = c.ContactID
	}

	return id, id != ""
}

// phoneDigits keeps only the digits of a number.
func phoneDigits(raw string) string {
	var b strings.Builder

	for _, r := range raw {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}

	return b.String()
}

// phoneMatches compares a stored CRM number with the wanted digits. With
// lastN > 0 the last N digits must be equal (anchored at the end). Without it
// the shorter number must be the tail of the longer one and hold at least 7
// digits, so +380670000001 matches 0670000001 but not 71234.
func phoneMatches(stored, wantedDigits string, lastN int) bool {
	have := phoneDigits(stored)
	if have == "" || wantedDigits == "" {
		return false
	}

	if lastN > 0 {
		return len(have) >= lastN && len(wantedDigits) >= lastN &&
			have[len(have)-lastN:] == wantedDigits[len(wantedDigits)-lastN:]
	}

	if have == wantedDigits {
		return true
	}

	short, long := have, wantedDigits
	if len(short) > len(long) {
		short, long = long, short
	}

	return len(short) >= minSuffixDigits && strings.HasSuffix(long, short)
}

// callerMatches compares a stored CRM number with the caller's digits: equal,
// or the shorter holds at least 7 digits and both end with the same
// min(9, len(shorter)) digits. So +380670000001 matches 0670000001 and
// +44 20 7946 0958 matches 020 7946 0958 (trunk prefix 0), while 71234 or a
// number that only contains the caller's digits in the middle does not.
func callerMatches(stored, callerDigits string) bool {
	have := phoneDigits(stored)
	if have == "" || callerDigits == "" {
		return false
	}

	if have == callerDigits {
		return true
	}

	n := min(len(have), len(callerDigits), nationalDigits)

	return n >= minSuffixDigits && have[len(have)-n:] == callerDigits[len(callerDigits)-n:]
}

// callerTrust says how a stored CRM number matches the caller's digits:
// verified when it is the caller's number — the same digits, or its local form
// with a trunk prefix (020 7946 0958 for +44 20 7946 0958, 0670000001 for
// +380670000001); unverified when they only share their tail (callerMatches);
// empty when they do not match.
//
// The number without its country code and without a trunk 0 (670000001 for
// +380670000001), and an international prefix 00, count as the same number
// too: the call auto-link always bound those, and a call's number is the
// caller's own (verified). This is wider than a chat's WhatsApp number, which
// always arrives in full international form: a call's number arrives in
// whatever form the trunk sends. Only a shorter shared tail stays a possible
// match.
func callerTrust(stored, callerDigits string) string {
	have, caller := withoutIntlPrefix(phoneDigits(stored)), withoutIntlPrefix(callerDigits)
	if !callerMatches(have, caller) {
		return ""
	}

	if have == caller || isLocalForm(have, caller) || isLocalForm(caller, have) ||
		isNationalForm(have, caller) || isNationalForm(caller, have) {
		return model.IdentificationTrustVerified
	}

	return model.IdentificationTrustUnverified
}

// withoutIntlPrefix drops the international call prefix 00 (0038067… is
// +38067…) when a whole international number follows it; a short number that
// merely starts with zeros is left as it is.
func withoutIntlPrefix(digits string) string {
	if rest, ok := strings.CutPrefix(digits, "00"); ok && len(rest) >= minIntlDigits {
		return rest
	}

	return digits
}

// isNationalForm reports whether national is long without its country code:
// at least nationalDigits digits, no trunk 0, the tail of long.
func isNationalForm(national, long string) bool {
	return len(national) < len(long) && len(national) >= nationalDigits &&
		!strings.HasPrefix(national, "0") && strings.HasSuffix(long, national)
}

// isLocalForm reports whether local is long written with a trunk prefix 0
// instead of its country code.
func isLocalForm(local, long string) bool {
	if len(local) >= len(long) || !strings.HasPrefix(local, "0") {
		return false
	}

	national := local[1:]

	return len(national) >= minLocalNationalDigits && strings.HasSuffix(long, national)
}

// emailMatches compares emails case-insensitively.
func emailMatches(stored, wanted string) bool {
	stored, wanted = strings.TrimSpace(stored), strings.TrimSpace(wanted)

	return stored != "" && strings.EqualFold(stored, wanted)
}

// substringTerm turns a client-entered value into a plain CRM substring term:
// the CRM splits terms on "*", reads "%" as a wildcard, a leading "=" as an
// exact match and "/x/" as a regular expression, so all are dropped; so is
// "\\", LIKE's escape character (a trailing one fails the CRM query).
func substringTerm(v string) string {
	v = strings.NewReplacer("*", "", "%", "", "\\", "").Replace(strings.TrimSpace(v))
	v = strings.TrimLeft(v, "=/")
	v = strings.TrimRight(v, "/")

	return strings.TrimSpace(v)
}

// isSearchableKey reports whether an attribute key can be sent to the CRM
// search as "key=value": non-empty, bounded, and free of the characters the
// CRM reads as search operators ("*", "%", "\\", "=", a leading "/"). Any other
// name a CRM variable may carry, spaces included, is accepted.
func isSearchableKey(key string) bool {
	if key == "" || utf8.RuneCountInString(key) > maxKeyLen || strings.HasPrefix(key, "/") {
		return false
	}

	return !strings.ContainsAny(key, "*%\\=")
}

func scalarString(v *structpb.Value) string {
	switch k := v.GetKind().(type) {
	case *structpb.Value_StringValue:
		return k.StringValue
	case *structpb.Value_NumberValue:
		return strconv.FormatFloat(k.NumberValue, 'f', -1, 64)
	case *structpb.Value_BoolValue:
		return strconv.FormatBool(k.BoolValue)
	default:
		return ""
	}
}
