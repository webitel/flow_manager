package contacts

import (
	"context"
	"fmt"
	"testing"

	contacts2 "github.com/webitel/flow_manager/api/gen/contacts"
	domcontacts "github.com/webitel/flow_manager/internal/domain/contacts"
	"github.com/webitel/flow_manager/internal/runtime/ops"
	"github.com/webitel/flow_manager/internal/runtime/tree"
)

// ── fakeContactClient ─────────────────────────────────────────────────────────

type fakeContactClient struct {
	locateResult *contacts2.Contact
	locateErr    error
	searchResult *contacts2.ContactList
	searchErr    error
	createResult *contacts2.Contact
	createErr    error
	updateResult *contacts2.Contact
	updateErr    error
	mergeVarsResult *contacts2.VariableList
	mergeVarsErr    error
	mergePhonesResult *contacts2.PhoneList
	mergePhonesErr    error
}

func (f *fakeContactClient) Locate(_ context.Context, _ string, _ *contacts2.LocateContactRequest) (*contacts2.Contact, error) {
	return f.locateResult, f.locateErr
}
func (f *fakeContactClient) Search(_ context.Context, _ string, _ *contacts2.SearchContactsRequest) (*contacts2.ContactList, error) {
	return f.searchResult, f.searchErr
}
func (f *fakeContactClient) SearchNA(_ context.Context, _ *contacts2.SearchContactsNARequest) (*contacts2.ContactList, error) {
	return f.searchResult, f.searchErr
}
func (f *fakeContactClient) Create(_ context.Context, _ string, _ *contacts2.InputContactRequest) (*contacts2.Contact, error) {
	return f.createResult, f.createErr
}
func (f *fakeContactClient) Update(_ context.Context, _ string, _ *contacts2.InputContactRequest) (*contacts2.Contact, error) {
	return f.updateResult, f.updateErr
}
func (f *fakeContactClient) MergeVariables(_ context.Context, _ string, _ *contacts2.MergeVariablesRequest) (*contacts2.VariableList, error) {
	return f.mergeVarsResult, f.mergeVarsErr
}
func (f *fakeContactClient) MergePhones(_ context.Context, _ string, _ *contacts2.MergePhonesRequest) (*contacts2.PhoneList, error) {
	return f.mergePhonesResult, f.mergePhonesErr
}

var _ domcontacts.Client = (*fakeContactClient)(nil)

// ── helpers ───────────────────────────────────────────────────────────────────

func contactInput(args map[string]any) ops.OpInput {
	return ops.OpInput{Node: &tree.Node{Args: args}, DomainID: 1}
}

func okClient() *fakeContactClient {
	return &fakeContactClient{
		locateResult:      &contacts2.Contact{},
		searchResult:      &contacts2.ContactList{},
		createResult:      &contacts2.Contact{},
		updateResult:      &contacts2.Contact{},
		mergeVarsResult:   &contacts2.VariableList{},
		mergePhonesResult: &contacts2.PhoneList{},
	}
}

// ── getContact ────────────────────────────────────────────────────────────────

func TestGetContact_NoToken(t *testing.T) {
	op := &getContactOp{client: okClient()}
	_, err := op.Execute(context.Background(), contactInput(map[string]any{
		"setVar": "result", "id": "abc",
	}))
	if err == nil {
		t.Fatal("expected error when token is missing")
	}
}

func TestGetContact_NoSetVar(t *testing.T) {
	op := &getContactOp{client: okClient()}
	_, err := op.Execute(context.Background(), contactInput(map[string]any{
		"token": "tok", "id": "abc",
	}))
	if err == nil {
		t.Fatal("expected error when setVar is missing")
	}
}

func TestGetContact_NoEtag(t *testing.T) {
	op := &getContactOp{client: okClient()}
	_, err := op.Execute(context.Background(), contactInput(map[string]any{
		"token": "tok", "setVar": "result",
	}))
	if err == nil {
		t.Fatal("expected error when etag/id is missing")
	}
}

func TestGetContact_Success(t *testing.T) {
	op := &getContactOp{client: okClient()}
	out, err := op.Execute(context.Background(), contactInput(map[string]any{
		"token": "tok", "setVar": "result", "id": "contact-1",
	}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := out.SetVars["result"]; !ok {
		t.Error("expected SetVars[result] to be set")
	}
}

func TestGetContact_DepError(t *testing.T) {
	cli := &fakeContactClient{locateErr: fmt.Errorf("grpc error")}
	op := &getContactOp{client: cli}
	_, err := op.Execute(context.Background(), contactInput(map[string]any{
		"token": "tok", "setVar": "result", "id": "contact-1",
	}))
	if err == nil {
		t.Fatal("expected error when Locate fails")
	}
}

// ── findContact ───────────────────────────────────────────────────────────────

func TestFindContact_NoToken(t *testing.T) {
	op := &findContactOp{client: okClient()}
	_, err := op.Execute(context.Background(), contactInput(map[string]any{"setVar": "r"}))
	if err == nil {
		t.Fatal("expected error when token is missing")
	}
}

func TestFindContact_NoSetVar(t *testing.T) {
	op := &findContactOp{client: okClient()}
	_, err := op.Execute(context.Background(), contactInput(map[string]any{"token": "tok"}))
	if err == nil {
		t.Fatal("expected error when setVar is missing")
	}
}

func TestFindContact_Success(t *testing.T) {
	op := &findContactOp{client: okClient()}
	out, err := op.Execute(context.Background(), contactInput(map[string]any{
		"token": "tok", "setVar": "found", "q": "+380XXXXXXXXX",
	}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := out.SetVars["found"]; !ok {
		t.Error("expected SetVars[found] to be set")
	}
}

func TestFindContact_DepError(t *testing.T) {
	cli := &fakeContactClient{searchErr: fmt.Errorf("timeout")}
	op := &findContactOp{client: cli}
	_, err := op.Execute(context.Background(), contactInput(map[string]any{
		"token": "tok", "setVar": "found",
	}))
	if err == nil {
		t.Fatal("expected error when Search fails")
	}
}

// ── updateContact ─────────────────────────────────────────────────────────────

func TestUpdateContact_NoToken(t *testing.T) {
	op := &updateContactOp{client: okClient()}
	_, err := op.Execute(context.Background(), contactInput(map[string]any{"setVar": "r"}))
	if err == nil {
		t.Fatal("expected error when token is missing")
	}
}

func TestUpdateContact_Success(t *testing.T) {
	op := &updateContactOp{client: okClient()}
	out, err := op.Execute(context.Background(), contactInput(map[string]any{
		"token":       "tok",
		"setVar":      "updated",
		"x_json_mask": []any{"variables"},
		"input":       map[string]any{"etag": "c-1"},
	}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := out.SetVars["updated"]; !ok {
		t.Error("expected SetVars[updated] to be set")
	}
}

func TestUpdateContact_DepError(t *testing.T) {
	cli := &fakeContactClient{updateErr: fmt.Errorf("permission denied")}
	op := &updateContactOp{client: cli}
	_, err := op.Execute(context.Background(), contactInput(map[string]any{
		"token": "tok", "setVar": "r",
	}))
	if err == nil {
		t.Fatal("expected error when Update fails")
	}
}
