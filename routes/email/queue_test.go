package email

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/webitel/wlog"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/webitel/flow_manager/flow"
	"github.com/webitel/flow_manager/gen/cc"
	"github.com/webitel/flow_manager/model"
	"github.com/webitel/flow_manager/providers/emailevent"
)

type fakeEventConn struct {
	model.EmailEventConnection
	vars model.Variables
	from string
}

func (c *fakeEventConn) EventId() string            { return "7c9e6679-7425-40de-944b-e07fc1f90ae7" }
func (c *fakeEventConn) ThreadId() int64            { return 42 }
func (c *fakeEventConn) DomainId() int64            { return 1 }
func (c *fakeEventConn) Log() *wlog.Logger          { return testLog }
func (c *fakeEventConn) InitialFromAddress() string { return c.from }
func (c *fakeEventConn) NodeId() string             { return "flow-1" }
func (c *fakeEventConn) Get(name string) (string, bool) {
	v, ok := c.vars[name]
	if !ok {
		return "", false
	}
	return fmt.Sprint(v), ok
}
func (c *fakeEventConn) ParseText(text string, ops ...model.ParseOption) string {
	return model.ParseText(c, text, ops...)
}
func (c *fakeEventConn) Set(_ context.Context, vars model.Variables) (model.Response, *model.AppError) {
	c.vars = vars
	return model.CallResponseOK, nil
}

type fakeJoiner struct {
	errs         []error
	calls        []*cc.EmailJoinToQueueRequest
	res          *cc.EmailJoinToQueueResponse
	set          bool // res is returned as is, even nil
	agentId      *int32
	lookupErr    *model.AppError
	lookups      int
	lookupDomain int64
}

func (j *fakeJoiner) FindQueueByName(domainId int64, name string) (int32, *model.AppError) {
	j.lookups++
	j.lookupDomain = domainId
	if name == "unknown" {
		return 0, nil // as the store answers for a name it does not find
	}
	return 3, nil
}

func (j *fakeJoiner) GetAgentIdByExtension(domainId int64, _ string) (*int32, *model.AppError) {
	j.lookups++
	j.lookupDomain = domainId
	return j.agentId, j.lookupErr
}

func queueArgs(id int32) QueueParameters {
	var args QueueParameters
	args.Queue.Id = id
	return args
}

func (j *fakeJoiner) EmailJoinToQueue(_ context.Context, in *cc.EmailJoinToQueueRequest) (*cc.EmailJoinToQueueResponse, error) {
	j.calls = append(j.calls, in)
	if len(j.errs) >= len(j.calls) {
		return nil, j.errs[len(j.calls)-1]
	}
	if j.set {
		return j.res, nil
	}
	return &cc.EmailJoinToQueueResponse{AttemptId: 7, QueueId: 3, Result: cc.EmailJoinToQueueResponse_RESULT_REPLAYED}, nil
}

var testLog = wlog.NewLogger(&wlog.LoggerConfiguration{})

var testRetry = joinRetry{attempts: 3, callTimeout: time.Second, budget: 5 * time.Second, baseDelay: time.Millisecond, maxDelay: time.Millisecond}

func TestJoinEmailQueueRetriesUnavailable(t *testing.T) {
	conn := &fakeEventConn{from: " Customer@Example.COM "}
	cli := &fakeJoiner{errs: []error{status.Error(codes.Unavailable, "down"), status.Error(codes.DeadlineExceeded, "slow")}}
	args := queueArgs(3)
	args.Priority, args.ExtraEmailCount = 20, true

	if _, err := joinEmailQueue(context.Background(), cli, conn, args, testRetry); err != nil {
		t.Fatal(err)
	}
	if len(cli.calls) != 3 {
		t.Fatalf("calls = %d, want 3", len(cli.calls))
	}
	req := cli.calls[0]
	if req.EmailEventId != conn.EventId() || req.ThreadId != 42 || req.DomainId != 1 || req.QueueId != 3 {
		t.Fatalf("request = %+v", req)
	}
	if req.Priority != 20 || !req.ExtraEmailCount || req.FromAddress != "customer@example.com" {
		t.Fatalf("routing parameters = %+v", req)
	}
	for _, retry := range cli.calls {
		if retry != req {
			t.Fatal("retry rebuilt the request")
		}
	}
	if conn.vars["cc_attempt_id"] != "7" || conn.vars["cc_queue_id"] != "3" || conn.vars["cc_join_result"] != "replayed" {
		t.Fatalf("vars = %v", conn.vars)
	}
}

func TestJoinEmailQueueDoesNotRetryPermanentError(t *testing.T) {
	cli := &fakeJoiner{errs: []error{status.Error(codes.FailedPrecondition, "disabled")}}

	_, err := joinEmailQueue(context.Background(), cli, &fakeEventConn{}, queueArgs(3), testRetry)
	if err == nil || err.Id != "email.join_queue.queue_precondition" || len(cli.calls) != 1 {
		t.Fatalf("err = %v, calls = %d", err, len(cli.calls))
	}
}

func TestJoinQueueRejectsLegacyConnection(t *testing.T) {
	var legacy model.EmailConnection
	_, err := (&Router{}).joinQueue(context.Background(), nil, legacy, nil)
	if err == nil || err.Id != "email.join_queue.not_supported" {
		t.Fatalf("err = %v", err)
	}
}

func TestJoinEmailQueueRejectsInvalidResponse(t *testing.T) {
	cases := map[string]*cc.EmailJoinToQueueResponse{
		"nil":            nil,
		"unspecified":    {AttemptId: 7, QueueId: 3},
		"unknown result": {AttemptId: 7, QueueId: 3, Result: 99},
		"no attempt":     {QueueId: 3, Result: cc.EmailJoinToQueueResponse_RESULT_CREATED},
		"no queue":       {AttemptId: 7, Result: cc.EmailJoinToQueueResponse_RESULT_CREATED},
	}
	for name, res := range cases {
		conn := &fakeEventConn{}
		_, err := joinEmailQueue(context.Background(), &fakeJoiner{res: res, set: true}, conn, queueArgs(3), testRetry)
		if err == nil || err.Id != "email.join_queue.invalid_response" || conn.vars != nil {
			t.Errorf("%s: err = %v, vars = %v", name, err, conn.vars)
		}
	}
}

func TestJoinEmailQueuePriorityValidation(t *testing.T) {
	for _, tc := range []struct {
		priority any
		valid    bool
		want     int32
	}{
		{priority: nil, valid: true}, {priority: 0, valid: true}, {priority: 100, valid: true, want: 100},
		{priority: "", valid: true}, {priority: " \t ", valid: true},
		{priority: "20", valid: true, want: 20}, {priority: -1}, {priority: 101},
		{priority: 1.5}, {priority: float64(4294967296)},
		{priority: "NaN"}, {priority: "Inf"}, {priority: "invalid"},
	} {
		for _, priority := range []any{tc.priority, "${priority}"} {
			if tc.priority == nil && priority != nil {
				continue
			}
			conn := &fakeEventConn{vars: model.Variables{"priority": tc.priority}}
			var args QueueParameters
			scope := &flow.Flow{Connection: conn}
			if err := scope.Decode(map[string]any{"queue": map[string]any{"id": 3}, "priority": priority}, &args); err != nil {
				t.Fatal(err)
			}
			cli := &fakeJoiner{}
			_, err := joinEmailQueue(context.Background(), cli, conn, args, testRetry)
			if tc.valid {
				if err != nil || len(cli.calls) != 1 || cli.calls[0].Priority != tc.want || cli.calls[0].StickyAgentId != 0 || cli.calls[0].ExtraEmailCount {
					t.Fatalf("priority %v: err=%v calls=%v", tc.priority, err, cli.calls)
				}
			} else if err == nil || err.Id != "email.join_queue.valid.priority" || len(cli.calls) != 0 || cli.lookups != 0 {
				t.Fatalf("invalid priority %v reached lookup/RPC: err=%v", tc.priority, err)
			}
		}
	}
}

func TestJoinEmailQueueEventVariables(t *testing.T) {
	for _, tc := range []struct {
		name     string
		vars     model.Variables
		priority int32
		agent    int32
		extra    bool
	}{
		{name: "missing variables"},
		{name: "empty variables", vars: model.Variables{"prio": "", "sticky": "", "extra": ""}},
		{name: "string variables", vars: model.Variables{"prio": "5", "sticky": "9", "extra": "true"}, priority: 5, agent: 9, extra: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			flowId := int64(1)
			conn := emailevent.NewConnection(context.Background(), "flow-1", &emailevent.InboundMessage{
				EventId: "7c9e6679-7425-40de-944b-e07fc1f90ae7", DomainId: 1, ThreadId: 42, MessageId: 7, FlowId: &flowId,
			}, testLog)
			defer conn.Close()
			conn.Set(context.Background(), tc.vars)
			scope := &flow.Flow{Connection: conn}
			var args QueueParameters
			if err := scope.Decode(map[string]any{
				"queue": map[string]any{"id": 3}, "priority": "${prio}",
				"agent": map[string]any{"id": "${sticky}"}, "extraEmailCount": "${extra}",
			}, &args); err != nil {
				t.Fatal(err)
			}
			cli := &fakeJoiner{}
			if _, err := joinEmailQueue(context.Background(), cli, conn, args, testRetry); err != nil {
				t.Fatal(err)
			}
			if len(cli.calls) != 1 || cli.lookups != 0 {
				t.Fatalf("calls=%d, lookups=%d", len(cli.calls), cli.lookups)
			}
			req := cli.calls[0]
			if req.Priority != tc.priority || req.StickyAgentId != tc.agent || req.ExtraEmailCount != tc.extra {
				t.Fatalf("routing values: %+v", req)
			}
		})
	}
}

func TestJoinEmailQueueAgentLookup(t *testing.T) {
	id := int32(8)
	lookupErr := model.NewAppError("test", "test.lookup", nil, "db", http.StatusInternalServerError)
	for _, tc := range []struct {
		input     string
		lookupId  *int32
		lookupErr *model.AppError
		want      int32
		lookups   int
		invalid   bool
	}{
		{input: `{"queue":{"id":3},"agent":{"id":9,"extension":"123"}}`, want: 9},
		{input: `{"queue":{"id":3},"agent":{"id":0,"extension":"123"}}`},
		{input: `{"queue":{"id":3},"agent":{"id":"","extension":"123"}}`},
		{input: `{"queue":{"id":3},"agent":{"extension":"123"}}`, lookupId: &id, want: 8, lookups: 1},
		{input: `{"queue":{"id":3},"agent":{"extension":"missing"}}`, lookups: 1},
		{input: `{"queue":{"id":3},"agent":{"extension":"123"}}`, lookupErr: lookupErr, lookups: 1},
		{input: `{"queue":{"id":3},"agent":{"id":-1}}`, invalid: true},
		{input: `{"queue":{"id":3},"agent":{"id":4294967297}}`, invalid: true},
	} {
		var args QueueParameters
		if err := json.Unmarshal([]byte(tc.input), &args); err != nil {
			t.Fatal(err)
		}
		cli := &fakeJoiner{agentId: tc.lookupId, lookupErr: tc.lookupErr}
		_, err := joinEmailQueue(context.Background(), cli, &fakeEventConn{}, args, testRetry)
		if tc.lookupErr != nil || tc.invalid {
			if err == nil || len(cli.calls) != 0 {
				t.Fatalf("error ignored: %s err=%v", tc.input, err)
			}
			if tc.lookupErr != nil && err != tc.lookupErr {
				t.Fatal("lookup error replaced by no candidate")
			}
		} else if err != nil || len(cli.calls) != 1 || cli.calls[0].StickyAgentId != tc.want {
			t.Fatalf("agent %s: err=%v calls=%v", tc.input, err, cli.calls)
		}
		if cli.lookups != tc.lookups || (cli.lookups != 0 && cli.lookupDomain != 1) {
			t.Fatal("lookup not scoped to event domain")
		}
	}
}

// fakeRoutes answers BeginFlowRoute with a fixed route and records the writes after the join.
type fakeRoutes struct {
	began   bool
	route   *model.EmailRoute
	err     *model.AppError
	finish  []model.EmailRouteResult
	retries int
	intent  *model.EmailRouteIntent
}

func (f *fakeRoutes) Route(context.Context, string) (*model.EmailRoute, *model.AppError) {
	return f.route, f.err
}

func (f *fakeRoutes) BeginFlowRoute(_ context.Context, _, _ string, intent model.EmailRouteIntent, _ time.Duration) (bool, *model.EmailRoute, *model.AppError) {
	f.intent = &intent
	return f.began, f.route, f.err
}

func (f *fakeRoutes) FinishRoute(_ context.Context, _, _ string, _ model.EmailRouteState, res model.EmailRouteResult) (bool, *model.AppError) {
	f.finish = append(f.finish, res)
	return true, nil
}

func (f *fakeRoutes) RetryRoute(context.Context, string, string, model.EmailRouteState, string, time.Time, bool) (bool, *model.AppError) {
	f.retries++
	return true, nil
}

// Repeated and late joins of a routed event follow the route, never the deadline alone.
func TestJoinEmailRoute(t *testing.T) {
	flowQueue, otherQueue := int32(5), int32(9)
	unavailable := status.Error(codes.Unavailable, "down")
	tests := []struct {
		name    string
		routes  *fakeRoutes
		cli     *fakeJoiner
		handled bool
		errId   string
		calls   int
		finish  model.EmailRouteState
		retries int
	}{
		{name: "an untracked event joins as before",
			routes: &fakeRoutes{route: &model.EmailRoute{State: model.EmailRouteNotNeeded}}, cli: &fakeJoiner{}},
		{name: "a row written before routing joins as before",
			routes: &fakeRoutes{route: &model.EmailRoute{State: model.EmailRouteUnset}}, cli: &fakeJoiner{}},
		{name: "a route removed by cleanup closes the window",
			routes: &fakeRoutes{}, cli: &fakeJoiner{}, handled: true, errId: "email_route.window_closed"},
		{name: "the intent is joined and recorded with the answered queue",
			routes: &fakeRoutes{began: true, route: &model.EmailRoute{State: model.EmailRouteJoining, QueueId: &flowQueue}},
			cli:    &fakeJoiner{}, handled: true, calls: 1, finish: model.EmailRouteRouted},
		{name: "an unusable queue drops the intent",
			routes:  &fakeRoutes{began: true, route: &model.EmailRoute{State: model.EmailRouteJoining, QueueId: &flowQueue}},
			cli:     &fakeJoiner{errs: []error{status.Error(codes.NotFound, "store.sql_member.email_join.queue_not_found")}},
			handled: true, errId: "email.join_queue.queue_not_found", calls: 1, finish: model.EmailRoutePending},
		{name: "an unknown outcome keeps the intent for the watcher",
			routes:  &fakeRoutes{began: true, route: &model.EmailRoute{State: model.EmailRouteJoining, QueueId: &flowQueue}},
			cli:     &fakeJoiner{errs: []error{unavailable, unavailable, unavailable}},
			handled: true, errId: "email.join_queue.unavailable", calls: 3, retries: 1},
		{name: "another queue conflicts with the intent",
			routes: &fakeRoutes{route: &model.EmailRoute{State: model.EmailRouteJoining, QueueId: &otherQueue}},
			cli:    &fakeJoiner{}, handled: true, errId: "email_route.queue_conflict"},
		{name: "a route of the flow is answered again",
			routes: &fakeRoutes{route: &model.EmailRoute{State: model.EmailRouteRouted, Source: model.EmailRouteSourceFlow, QueueId: &otherQueue}},
			cli:    &fakeJoiner{}, handled: true, calls: 1},
		{name: "the fallback route closes the window",
			routes: &fakeRoutes{route: &model.EmailRoute{State: model.EmailRouteRouted, Source: model.EmailRouteSourceFallback, QueueId: &otherQueue}},
			cli:    &fakeJoiner{}, handled: true, errId: "email_route.window_closed"},
		{name: "a pending route past the deadline closes the window",
			routes: &fakeRoutes{route: &model.EmailRoute{State: model.EmailRoutePending, DuePassed: true}},
			cli:    &fakeJoiner{}, handled: true, errId: "email_route.window_closed"},
		{name: "resolving closes the window",
			routes: &fakeRoutes{route: &model.EmailRoute{State: model.EmailRouteResolving}},
			cli:    &fakeJoiner{}, handled: true, errId: "email_route.window_closed"},
		{name: "an intent that cannot be written is not joined",
			routes: &fakeRoutes{err: model.NewAppError("test", "test", nil, "db", 500)},
			cli:    &fakeJoiner{}, handled: true, errId: "email_route.unavailable"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conn := &fakeEventConn{}
			req := &cc.EmailJoinToQueueRequest{QueueId: flowQueue, ThreadId: 42, EmailEventId: conn.EventId(), DomainId: 1,
				Priority: 20, StickyAgentId: 8, ExtraEmailCount: true, FromAddress: "c@example.com"}
			handled, _, err := joinEmailRoute(context.Background(), tt.routes, tt.cli, conn, req, testRetry)

			errId := ""
			if err != nil {
				errId = err.Id
			}
			if handled != tt.handled || errId != tt.errId || len(tt.cli.calls) != tt.calls || tt.routes.retries != tt.retries {
				t.Fatalf("handled %v, err %q, calls %d, retries %d", handled, errId, len(tt.cli.calls), tt.routes.retries)
			}
			if tt.finish == "" && len(tt.routes.finish) != 0 || tt.finish != "" && (len(tt.routes.finish) != 1 || tt.routes.finish[0].State != tt.finish) {
				t.Fatalf("finish %+v, want %s", tt.routes.finish, tt.finish)
			}
			if tt.routes.intent != nil && *tt.routes.intent != (model.EmailRouteIntent{QueueId: flowQueue, Priority: 20, StickyAgentId: 8, ExtraEmailCount: true}) {
				t.Fatalf("intent %+v", *tt.routes.intent)
			}
			// Every call carries the routing values frozen before the route.
			for _, c := range tt.cli.calls {
				if c.Priority != 20 || c.StickyAgentId != 8 || !c.ExtraEmailCount || c.FromAddress != "c@example.com" {
					t.Fatalf("routing values lost: %+v", c)
				}
			}
			if tt.finish == model.EmailRouteRouted && (*tt.routes.finish[0].QueueId != 3 || conn.vars["cc_queue_id"] != "3") {
				t.Fatalf("routed %+v, vars %v", tt.routes.finish[0], conn.vars)
			}
			if tt.calls == 1 && tt.errId == "" && tt.finish == "" &&
				(tt.cli.calls[0].QueueId != otherQueue || conn.vars["cc_attempt_id"] != "7" ||
					conn.vars["cc_queue_id"] != "3" || conn.vars["cc_join_result"] != "replayed") {
				t.Fatalf("replay request %+v, vars %v", tt.cli.calls[0], conn.vars)
			}
		})
	}
}

// A repeated JoinQueue is settled by the stored route before any agent lookup.
func TestStoredFlowRequest(t *testing.T) {
	queue, other := int32(3), int32(5) // fakeJoiner resolves a queue name to 3, "unknown" to 0
	in := &model.EmailRouteIntent{QueueId: queue, Priority: 20, StickyAgentId: 8, ExtraEmailCount: true}
	ext := "2002"
	args := func(id int32, name string, priority any) QueueParameters {
		p := QueueParameters{Priority: priority, Agent: &struct {
			Id        any     `json:"id"`
			Extension *string `json:"extension"`
		}{Extension: &ext}}
		p.Queue.Id, p.Queue.Name = id, name
		return p
	}
	byName, unknown, otherId := args(0, "support", nil), args(0, "unknown", nil), args(9, "", nil)
	route := func(state model.EmailRouteState, source string, in *model.EmailRouteIntent) *model.EmailRoute {
		return &model.EmailRoute{State: state, Source: source, QueueId: &in.QueueId, Intent: in}
	}
	joining := route(model.EmailRouteJoining, model.EmailRouteSourceFlow, in)
	routed := route(model.EmailRouteRouted, model.EmailRouteSourceFlow, &model.EmailRouteIntent{QueueId: other, Priority: 20, StickyAgentId: 8, ExtraEmailCount: true})
	late := &model.EmailRoute{State: model.EmailRoutePending, DuePassed: true, Intent: &model.EmailRouteIntent{}}

	cases := []struct {
		name    string
		route   *model.EmailRoute
		argv    QueueParameters
		kept    bool
		lookups int
		errId   string
	}{
		{"joining, same queue by name", joining, byName, true, 1, ""},
		{"joining, another queue by name", route(model.EmailRouteJoining, model.EmailRouteSourceFlow, &model.EmailRouteIntent{QueueId: other}), byName, false, 1, "email_route.queue_conflict"},
		{"joining, another queue id", joining, otherId, false, 0, "email_route.queue_conflict"},
		{"joining, queue name not found", joining, unknown, false, 1, "email.join_queue.valid.queue"},
		{"routed by the Flow, another queue", routed, byName, true, 0, ""},
		{"routed by the Flow, unknown name", routed, unknown, true, 0, ""},
		{"routed by the Flow, no queue", routed, args(0, "", nil), false, 0, "email.join_queue.valid.queue"},
		{"invalid priority in a repeat", joining, args(queue, "", "500"), false, 0, "email.join_queue.valid.priority"},
		{"routed by the fallback", route(model.EmailRouteRouted, model.EmailRouteSourceFallback, in), byName, false, 0, "email_route.window_closed"},
		{"pending after the deadline", late, unknown, false, 0, "email_route.window_closed"},
		{"pending before the deadline", &model.EmailRoute{State: model.EmailRoutePending, Intent: &model.EmailRouteIntent{}}, byName, false, 0, ""},
	}
	for _, c := range cases {
		conn := &fakeEventConn{from: "C@Example.com"}
		cli := &fakeJoiner{}
		req, err := storedFlowRequest(context.Background(), &fakeRoutes{route: c.route}, cli, conn, c.argv)
		if (err == nil) != (c.errId == "") || (err != nil && err.Id != c.errId) || (req != nil) != c.kept || cli.lookups != c.lookups {
			t.Fatalf("%s: req %+v, err %v, lookups %d", c.name, req, err, cli.lookups)
		}
		if c.kept && (req.QueueId != c.route.Intent.QueueId || req.Priority != 20 || req.StickyAgentId != 8 || !req.ExtraEmailCount ||
			req.FromAddress != "c@example.com" || req.EmailEventId != conn.EventId()) {
			t.Fatalf("%s: %+v", c.name, req)
		}
	}
}
