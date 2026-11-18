package model

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

// ContactIdentificationApp is the schema application id of the "Contact
// identification" node. It runs in chat and call schemas.
const ContactIdentificationApp = "contactIdentification"

// Trust values written to the schema; the spec names them in lower case.
const (
	IdentificationTrustVerified   = "verified"
	IdentificationTrustUnverified = "unverified"
)

// Criterion kinds sent to identification.
const (
	IdentificationCriterionPhone     = "phone"
	IdentificationCriterionEmail     = "email"
	IdentificationCriterionAttribute = "attribute"
)

// ContactIdentificationArgs are the node's arguments as stored in the schema.
// Keys are camelCase like every other flow application's arguments.
//
//nolint:tagliatelle // schema argument names, shared with the flow editor
type ContactIdentificationArgs struct {
	// ConversationId overrides the current chat (chat schemas only).
	ConversationID string `json:"conversationId"`
	// Variables are the criteria: key "phone" and "email" match phones and
	// emails, any other key a CRM contact variable of that name. Matched with OR.
	Variables []ContactIdentificationVariable `json:"variables"`
	// SetVar receives the summary as JSON: {found_count, contact_id, trust}.
	SetVar string `json:"setVar"`
	// PhoneLastDigits matches phones by their last N digits; 0 = whole number.
	PhoneLastDigits int `json:"phoneLastDigits"`
	// PhoneScope is "any" (default) or "primary" ("primary_only" and
	// "Primary only" are read as "primary" too).
	PhoneScope string `json:"phoneScope"`
}

type ContactIdentificationVariable struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// ContactIdentificationCriterion is one value to match.
type ContactIdentificationCriterion struct {
	Type  string
	Key   string
	Value string
}

// Limits of one identification run, the same as im-thread's API accepts: a
// criterion past them would fail the whole run there.
const (
	IdentificationMaxCriteria       = 16
	IdentificationMaxValueLen       = 256
	IdentificationMaxKeyLen         = 64
	IdentificationMinPhoneLastDigit = 4
	IdentificationMaxPhoneLastDigit = 15
)

// Criteria turns the node variables into criteria. Empty values (a variable
// the bot never collected), values or keys over the limits and criteria past
// the 16th are skipped, so one bad variable never fails the whole run.
func (a ContactIdentificationArgs) Criteria() []ContactIdentificationCriterion {
	out := make([]ContactIdentificationCriterion, 0, min(len(a.Variables), IdentificationMaxCriteria))

	for _, v := range a.Variables {
		if len(out) == IdentificationMaxCriteria {
			break
		}

		key, value := strings.TrimSpace(v.Key), strings.TrimSpace(v.Value)
		if key == "" || value == "" ||
			utf8.RuneCountInString(value) > IdentificationMaxValueLen || utf8.RuneCountInString(key) > IdentificationMaxKeyLen {
			continue
		}

		switch strings.ToLower(key) {
		case IdentificationCriterionPhone:
			out = append(out, ContactIdentificationCriterion{Type: IdentificationCriterionPhone, Value: value})
		case IdentificationCriterionEmail:
			out = append(out, ContactIdentificationCriterion{Type: IdentificationCriterionEmail, Value: value})
		default:
			out = append(out, ContactIdentificationCriterion{Type: IdentificationCriterionAttribute, Key: key, Value: value})
		}
	}

	return out
}

// PhoneLastDigitsOrWhole returns the configured last-N for phone matching, or
// 0 (the whole number) when it is unset or outside 4..15; ok is false when a
// configured value was dropped.
func (a ContactIdentificationArgs) PhoneLastDigitsOrWhole() (n int, ok bool) {
	if a.PhoneLastDigits == 0 {
		return 0, true
	}

	if a.PhoneLastDigits < IdentificationMinPhoneLastDigit || a.PhoneLastDigits > IdentificationMaxPhoneLastDigit {
		return 0, false
	}

	return a.PhoneLastDigits, true
}

// PrimaryPhonesOnly reports whether only primary CRM phones are matched. The
// editor's spelling of the option ("primary", "primary_only", "Primary only")
// does not matter; anything else, empty included, is "any".
func (a ContactIdentificationArgs) PrimaryPhonesOnly() bool {
	scope := strings.Map(func(r rune) rune {
		if r == '_' || r == '-' || unicode.IsSpace(r) {
			return -1
		}

		return unicode.ToLower(r)
	}, a.PhoneScope)

	return scope == "primary" || scope == "primaryonly"
}

// ContactIdentificationRequest is one identification run for a chat.
type ContactIdentificationRequest struct {
	ThreadID        string
	Criteria        []ContactIdentificationCriterion
	PhoneLastDigits int
	PrimaryOnly     bool
}

// ContactIdentificationResult is the summary written to the schema.
type ContactIdentificationResult struct {
	FoundCount int `json:"found_count"`
	// ContactID is set only when there is exactly one candidate, it is
	// verified and it is the contact bound to the call or chat.
	ContactID string `json:"contact_id,omitempty"`
	// Trust is the highest trust among the candidates; empty when none.
	Trust string `json:"trust,omitempty"`
}
