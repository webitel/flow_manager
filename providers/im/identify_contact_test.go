package im

import (
	"testing"

	t "github.com/webitel/flow_manager/gen/im/service/thread/v1"
	"github.com/webitel/flow_manager/model"
)

func TestIdentifyRequest(tt *testing.T) {
	tt.Parallel()

	req := model.ContactIdentificationRequest{
		PhoneLastDigits: 9,
		PrimaryOnly:     true,
		Criteria: []model.ContactIdentificationCriterion{
			{Type: model.IdentificationCriterionPhone, Value: "+380000000001"},
			{Type: model.IdentificationCriterionEmail, Value: "qa@example.com"},
			{Type: model.IdentificationCriterionAttribute, Key: "contract", Value: "A-1"},
		},
	}

	in := identifyRequest("019fcc11-0000-7000-8000-000000000001", 7, req)

	if in.GetThreadId() != "019fcc11-0000-7000-8000-000000000001" || in.GetDomainId() != 7 || in.GetPhoneLastDigits() != 9 ||
		in.GetPhoneScope() != t.PhoneScope_PHONE_SCOPE_PRIMARY_ONLY || len(in.GetCriteria()) != 3 {
		tt.Fatalf("request = %v", in)
	}

	if c := in.GetCriteria()[0]; c.GetType() != t.IdentificationCriterionType_IDENTIFICATION_CRITERION_TYPE_PHONE || c.GetKey() != "" {
		tt.Fatalf("phone criterion = %v", c)
	}

	if c := in.GetCriteria()[2]; c.GetType() != t.IdentificationCriterionType_IDENTIFICATION_CRITERION_TYPE_ATTRIBUTE || c.GetKey() != "contract" {
		tt.Fatalf("attribute criterion = %v", c)
	}
}

func TestIdentifyRequest_ConversationID(tt *testing.T) {
	tt.Parallel()

	const current, other = "019fcc11-0000-7000-8000-000000000001", "019fcc11-0000-7000-8000-000000000002"

	if got := identifyRequest(current, 1, model.ContactIdentificationRequest{}).GetThreadId(); got != current {
		tt.Fatalf("without conversationId the thread = %q, want the current chat %q", got, current)
	}

	if got := identifyRequest(current, 1, model.ContactIdentificationRequest{ThreadID: other}).GetThreadId(); got != other {
		tt.Fatalf("with conversationId the thread = %q, want %q", got, other)
	}

	if got := identifyRequest(current, 1, model.ContactIdentificationRequest{}).GetPhoneScope(); got != t.PhoneScope_PHONE_SCOPE_UNSPECIFIED {
		tt.Fatalf("phone scope = %v, want any by default", got)
	}
}

func TestIdentifyResult(tt *testing.T) {
	tt.Parallel()

	got := identifyResult(&t.IdentifyContactResponse{FoundCount: 1, CrmContactId: "10", Trust: t.IdentificationTrust_IDENTIFICATION_TRUST_VERIFIED})
	if got.FoundCount != 1 || got.ContactID != "10" || got.Trust != model.IdentificationTrustVerified {
		tt.Fatalf("result = %+v", got)
	}

	none := identifyResult(&t.IdentifyContactResponse{})
	if none.FoundCount != 0 || none.ContactID != "" || none.Trust != "" {
		tt.Fatalf("empty result = %+v", none)
	}

	if js := model.ToJson(none); js != `{"found_count":0}` {
		tt.Fatalf("variable = %s, want found_count only", js)
	}
}
