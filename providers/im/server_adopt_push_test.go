package im

import (
	"context"
	"slices"
	"testing"

	"google.golang.org/grpc"

	"github.com/webitel/engine/pkg/wbt"
	"github.com/webitel/wlog"

	p "github.com/webitel/flow_manager/gen/im/api/gateway/v1"
	"github.com/webitel/flow_manager/model"
)

const testThreadID = "85a27368-d03f-41e3-8f96-92d64c30c23e"

type fakeSessionStore struct{}

func (fakeSessionStore) Touch(string, string) (*int, error) {
	seq := 1
	return &seq, nil
}
func (fakeSessionStore) Remove(string, string) error { return nil }
func (fakeSessionStore) RemoveByThread(string) error { return nil }
func (fakeSessionStore) RemoveAll(string) error      { return nil }

type fakeGatewayThreads struct {
	p.ThreadManagementClient
}

func (fakeGatewayThreads) Search(context.Context, *p.ThreadSearchRequest, ...grpc.CallOption) (*p.SearchThreadResponse, error) {
	return &p.SearchThreadResponse{Items: []*p.Thread{{
		Id: testThreadID,
		Members: []*p.ThreadMember{
			{Id: "m-cust", Role: p.ThreadRole_ROLE_OWNER, Contact: &p.Contact{Sub: "cust-1", Iss: "webitel"}},
			{Id: "m-bot-1", Role: p.ThreadRole_ROLE_OWNER, Contact: &p.Contact{Sub: "1", Iss: IMUserTypeBot, IsBot: true}},
			{Id: "m-bot-145", Role: p.ThreadRole_ROLE_MEMBER, Contact: &p.Contact{Sub: "145", Iss: IMUserTypeBot, IsBot: true}},
		},
	}}}, nil
}

func newTestServer(tb testing.TB) *server {
	tb.Helper()

	log := wlog.NewLogger(&wlog.LoggerConfiguration{EnableConsole: false})
	wlog.InitGlobalLogger(log)

	return &server{
		id:              "fm-1",
		log:             log,
		connectionStore: NewConnectionStore(log),
		sessionStore:    fakeSessionStore{},
		consume:         make(chan model.Connection, 10),
		client: &Client{
			ctx:           context.Background(),
			threadService: &wbt.Client[p.ThreadManagementClient]{Api: fakeGatewayThreads{}},
		},
	}
}

func customerMessage() model.IMEventWrapper {
	return model.MessageWrapper[model.Message]{
		DomainID: 1,
		Type:     model.IMEventTypeMessage,
		Message: model.Message{
			ThreadID: testThreadID,
			DomainID: 1,
			Text:     "hello",
			From:     model.ImEndpoint{Sub: "cust-1", Issuer: "webitel"},
			To: []model.ImEndpoint{
				{Sub: "1", Issuer: IMUserTypeBot, MemberID: "m-bot-1"},
				{Sub: "145", Issuer: IMUserTypeBot, MemberID: "m-bot-145"},
			},
		},
	}
}

func dispatched(ch chan model.Connection) []string {
	var ids []string
	for {
		select {
		case c := <-ch:
			ids = append(ids, c.Id())
		default:
			return ids
		}
	}
}

func TestHandleBotControlGranted_PushAfterMessageStartedSchema(t *testing.T) {
	s := newTestServer(t)

	owner := newConnection(s, testThreadID+".1", model.ImEndpoint{Sub: "1", Issuer: IMUserTypeBot, MemberID: "m-bot-1"}, customerMessage(), false)
	s.connectionStore.Add(owner)

	if err := s.nodeMessage(customerMessage()); err != nil {
		t.Fatalf("nodeMessage: %v", err)
	}

	err := s.handleBotControlGranted(model.IMBotControlGrantedEvent{
		ThreadID:    testThreadID,
		DomainID:    1,
		MemberID:    "m-bot-145",
		AutoLeave:   true,
		ReleasedSub: 1,
		Sub:         145,
		Reason:      "transfer",
	})
	if err != nil {
		t.Fatalf("handleBotControlGranted: %v", err)
	}

	if got := dispatched(s.consume); !slices.Equal(got, []string{testThreadID + ".145"}) {
		t.Errorf("dispatched %v, want only the transfer bot", got)
	}
	if !owner.IsSuspended() {
		t.Error("owner bot is not suspended after the push grant")
	}
	if parent, ok := s.pushParent.Load(testThreadID + ".145"); !ok || parent != testThreadID+".1" {
		t.Errorf("push parent = %v, want %s", parent, testThreadID+".1")
	}
}

func TestHandleBotControlGranted_DuplicateGrantKeepsRunningSchema(t *testing.T) {
	s := newTestServer(t)

	owner := newConnection(s, testThreadID+".1", model.ImEndpoint{Sub: "1", Issuer: IMUserTypeBot, MemberID: "m-bot-1"}, customerMessage(), false)
	s.connectionStore.Add(owner)

	err := s.handleBotControlGranted(model.IMBotControlGrantedEvent{
		ThreadID: testThreadID,
		DomainID: 1,
		MemberID: "m-bot-1",
		Sub:      1,
		Reason:   "transfer",
	})
	if err != nil {
		t.Fatalf("handleBotControlGranted: %v", err)
	}

	if owner.IsSuspended() {
		t.Error("duplicate grant suspended the running schema")
	}
	if got := dispatched(s.consume); len(got) != 0 {
		t.Errorf("duplicate grant dispatched %v", got)
	}
}
