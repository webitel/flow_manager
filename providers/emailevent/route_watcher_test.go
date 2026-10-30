package emailevent

import (
	"context"
	"testing"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/webitel/flow_manager/gen/cc"
	"github.com/webitel/flow_manager/model"
	"github.com/webitel/flow_manager/store"
)

// watcherStore hands out fixed routes and records how each one ended.
type watcherStore struct {
	store.EmailEventStore
	routes     []*model.EmailRoute
	finish     []model.EmailRouteResult
	retries    []bool
	finishLost bool
}

func (s *watcherStore) ClaimRoutes(context.Context, string, int, time.Duration) ([]*model.EmailRoute, *model.AppError) {
	routes := s.routes
	s.routes = nil
	return routes, nil
}

func (s *watcherStore) FinishRoute(_ context.Context, _, _ string, _ model.EmailRouteState, res model.EmailRouteResult) (bool, *model.AppError) {
	s.finish = append(s.finish, res)
	return !s.finishLost, nil
}

func (s *watcherStore) RetryRoute(_ context.Context, _, _ string, _ model.EmailRouteState, _ string, _ time.Time, counted bool) (bool, *model.AppError) {
	s.retries = append(s.retries, counted)
	return true, nil
}

type watcherJoiner struct {
	err   error
	calls []*cc.EmailJoinToQueueRequest
}

func (j *watcherJoiner) EmailJoinToQueue(_ context.Context, in *cc.EmailJoinToQueueRequest) (*cc.EmailJoinToQueueResponse, error) {
	j.calls = append(j.calls, in)
	if j.err != nil {
		return nil, j.err
	}
	return &cc.EmailJoinToQueueResponse{AttemptId: 7, QueueId: 3, Result: cc.EmailJoinToQueueResponse_RESULT_CREATED}, nil
}

// The watcher joins only the stored queue and ends a route only on a certain outcome.
func TestRouteWatcherResolves(t *testing.T) {
	fallback, intent := int32(5), int32(9)
	tests := []struct {
		name    string
		route   model.EmailRoute
		err     error
		queue   int32 // 0: no join
		state   model.EmailRouteState
		source  string
		reason  string
		retried bool
	}{
		{name: "fallback after the deadline", route: model.EmailRoute{State: model.EmailRouteResolving, FallbackQueueId: &fallback},
			queue: 5, state: model.EmailRouteRouted, source: model.EmailRouteSourceFallback},
		{name: "an unfinished intent of the flow", route: model.EmailRoute{State: model.EmailRouteJoining, QueueId: &intent, FallbackQueueId: &fallback},
			queue: 9, state: model.EmailRouteRouted, source: model.EmailRouteSourceFlow},
		{name: "no fallback queue", route: model.EmailRoute{State: model.EmailRouteResolving},
			state: model.EmailRouteUnrouted, reason: RouteReasonNoFallbackQueue},
		{name: "an unusable fallback queue", route: model.EmailRoute{State: model.EmailRouteResolving, FallbackQueueId: &fallback},
			err: status.Error(codes.NotFound, joinQueueNotFound), queue: 5, state: model.EmailRouteUnrouted, reason: RouteReasonQueueNotFound},
		{name: "call_center unavailable is retried past the limit", route: model.EmailRoute{State: model.EmailRouteResolving, FallbackQueueId: &fallback, Attempts: 20},
			err: status.Error(codes.Unavailable, "down"), queue: 5, retried: true},
		{name: "an unexpected error is retried", route: model.EmailRoute{State: model.EmailRouteJoining, QueueId: &intent},
			err: status.Error(codes.Internal, "boom"), queue: 9, retried: true},
		{name: "an unexpected error at the limit fails", route: model.EmailRoute{State: model.EmailRouteJoining, QueueId: &intent, Attempts: 2},
			err: status.Error(codes.Internal, "boom"), queue: 9, state: model.EmailRouteFailed, reason: RouteReasonUnexpectedError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			route := tt.route
			route.EventId, route.DomainId, route.ThreadId = "0199b2a4-3c4e-7a10-8f2e-1b2c3d4e5f60", 1, 4
			st, cli := &watcherStore{routes: []*model.EmailRoute{&route}}, &watcherJoiner{err: tt.err}
			cfg := model.EmailEventsSettings{RouteWatcherBatch: 10, RouteLease: time.Minute, RouteRetryMaxInterval: time.Minute, RouteMaxAttempts: 3}
			w := newRouteWatcher(&Server{nodeId: "flow-1", runId: "run", cfg: cfg, store: st, joiner: cli, log: testLog})

			w.cycle(context.Background())

			switch {
			case tt.queue == 0 && len(cli.calls) != 0, tt.queue != 0 && (len(cli.calls) != 1 || cli.calls[0].QueueId != tt.queue):
				t.Fatalf("joins %v, want queue %d", cli.calls, tt.queue)
			case tt.retried && (len(st.retries) != 1 || !st.retries[0] || len(st.finish) != 0):
				t.Fatalf("retries %v, finish %+v", st.retries, st.finish)
			case !tt.retried && (len(st.finish) != 1 || st.finish[0].State != tt.state || st.finish[0].Source != tt.source || st.finish[0].Reason != tt.reason):
				t.Fatalf("finish %+v, want %s %s %q", st.finish, tt.state, tt.source, tt.reason)
			}
		})
	}
}

// Routes that cannot be joined before the lease ends go back without spending an attempt.
func TestRouteWatcherReturnsRoutesBeforeLeaseEnds(t *testing.T) {
	fallback := int32(5)
	st := &watcherStore{routes: []*model.EmailRoute{
		{EventId: "a", State: model.EmailRouteResolving, FallbackQueueId: &fallback},
		{EventId: "b", State: model.EmailRouteResolving, FallbackQueueId: &fallback},
	}}
	cli := &watcherJoiner{}
	cfg := model.EmailEventsSettings{RouteWatcherBatch: 10, RouteLease: routeJoinTimeout + routeWriteTimeout, RouteMaxAttempts: 3}
	w := newRouteWatcher(&Server{nodeId: "flow-1", runId: "run", cfg: cfg, store: st, joiner: cli, log: testLog})

	w.cycle(context.Background())

	if len(cli.calls) != 0 || len(st.retries) != 2 || st.retries[0] || st.retries[1] {
		t.Fatalf("joins %d, retries %v", len(cli.calls), st.retries)
	}
}

func TestRouteWatcherDoesNotCountLostFinish(t *testing.T) {
	reader := metric.NewManualReader()
	provider := metric.NewMeterProvider(metric.WithReader(reader))
	counter, err := provider.Meter("test.email.route").Int64Counter("test.email.route.outcome")
	if err != nil {
		t.Fatal(err)
	}
	previous := routeOutcomes
	routeOutcomes = counter
	t.Cleanup(func() {
		routeOutcomes = previous
		_ = provider.Shutdown(context.Background())
	})

	fallback := int32(5)
	st := &watcherStore{finishLost: true}
	cli := &watcherJoiner{}
	w := newRouteWatcher(&Server{nodeId: "flow-1", runId: "run", store: st, joiner: cli, log: testLog})
	route := &model.EmailRoute{
		EventId:  "0199b2a4-3c4e-7a10-8f2e-1b2c3d4e5f60",
		DomainId: 1, ThreadId: 4, State: model.EmailRouteResolving, FallbackQueueId: &fallback,
	}
	stats := &routeCycle{}

	w.resolve(context.Background(), route, stats)

	if len(cli.calls) != 1 || len(st.finish) != 1 || stats.routed != 0 || stats.fallback != 0 ||
		stats.unrouted != 0 || stats.failed != 0 {
		t.Fatalf("lost route finish counted: calls=%d finish=%d stats=%+v", len(cli.calls), len(st.finish), stats)
	}

	var before metricdata.ResourceMetrics
	if err := reader.Collect(context.Background(), &before); err != nil {
		t.Fatal(err)
	}
	if got := routeOutcomeCount(before, RouteOutcomeFallback); got != 0 {
		t.Fatalf("lost route finish emitted fallback metric: %d", got)
	}

	st.finishLost = false
	w.resolve(context.Background(), route, stats)
	var after metricdata.ResourceMetrics
	if err := reader.Collect(context.Background(), &after); err != nil {
		t.Fatal(err)
	}
	if got := routeOutcomeCount(after, RouteOutcomeFallback); got != 1 || stats.fallback != 1 {
		t.Fatalf("committed route counted %d times, stats=%+v", got, stats)
	}

	w.resolve(context.Background(), &model.EmailRoute{
		EventId:  "0199b2a4-3c4e-7a10-8f2e-1b2c3d4e5f61",
		DomainId: 1, ThreadId: 5, State: model.EmailRouteResolving,
	}, stats)
	cli.err = status.Error(codes.Internal, "boom")
	w.cfg.RouteMaxAttempts = 3
	intent := int32(9)
	w.resolve(context.Background(), &model.EmailRoute{
		EventId:  "0199b2a4-3c4e-7a10-8f2e-1b2c3d4e5f62",
		DomainId: 1, ThreadId: 6, State: model.EmailRouteJoining, QueueId: &intent, Attempts: 2,
	}, stats)
	var terminal metricdata.ResourceMetrics
	if err := reader.Collect(context.Background(), &terminal); err != nil {
		t.Fatal(err)
	}
	if routeOutcomeCount(terminal, RouteOutcomeUnrouted) != 1 || routeOutcomeCount(terminal, RouteOutcomeFailed) != 1 ||
		stats.unrouted != 1 || stats.failed != 1 {
		t.Fatalf("terminal outcomes were not counted once: stats=%+v", stats)
	}
}

func routeOutcomeCount(metrics metricdata.ResourceMetrics, outcome string) int64 {
	var count int64
	for _, scope := range metrics.ScopeMetrics {
		for _, m := range scope.Metrics {
			sum, ok := m.Data.(metricdata.Sum[int64])
			if !ok {
				continue
			}
			for _, point := range sum.DataPoints {
				if value, ok := point.Attributes.Value(attribute.Key("outcome")); ok && value.AsString() == outcome {
					count += point.Value
				}
			}
		}
	}
	return count
}

// The watcher repeats the Flow intent; a fallback or an older row uses the defaults; both keep From.
func TestRouteJoinRequest(t *testing.T) {
	intent := &model.EmailRouteIntent{QueueId: 5, Priority: 20, StickyAgentId: 8, ExtraEmailCount: true}
	base := model.EmailRoute{EventId: "e", DomainId: 1, ThreadId: 42, FromAddress: "c@example.com"}
	cases := []struct {
		name   string
		state  model.EmailRouteState
		intent *model.EmailRouteIntent
		want   [3]any
	}{
		{"flow intent", model.EmailRouteJoining, intent, [3]any{int32(20), int32(8), true}},
		{"fallback", model.EmailRouteResolving, nil, [3]any{int32(0), int32(0), false}},
		{"row written before the intent values", model.EmailRouteJoining, nil, [3]any{int32(0), int32(0), false}},
	}
	for _, c := range cases {
		route := base
		route.State, route.Intent = c.state, c.intent
		req := routeJoinRequest(&route, 7)
		got := [3]any{req.Priority, req.StickyAgentId, req.ExtraEmailCount}
		if got != c.want || req.QueueId != 7 || req.FromAddress != "c@example.com" || req.EmailEventId != "e" || req.ThreadId != 42 {
			t.Fatalf("%s: %+v", c.name, req)
		}
	}
}
