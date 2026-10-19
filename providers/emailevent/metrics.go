package emailevent

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/metric/noop"
)

// Route outcomes counted for operators; labels never carry event or Thread ids.
const (
	RouteOutcomeFallback     = "fallback"
	RouteOutcomeWindowClosed = "window_closed"
	RouteOutcomeUnrouted     = "unrouted"
	RouteOutcomeFailed       = "route_failed"
)

var routeOutcomes = newRouteOutcomes()

// newRouteOutcomes uses the global provider, which forwards to the one configured later.
func newRouteOutcomes() metric.Int64Counter {
	counter, err := otel.Meter("github.com/webitel/flow_manager/providers/emailevent").Int64Counter(
		"flow.email.route.outcome",
		metric.WithDescription("Email routes that used the fallback queue, came too late or ended without a queue"),
	)
	if err != nil {
		return noop.Int64Counter{}
	}

	return counter
}

// RecordRouteOutcome counts an outcome once it is committed or returned to the Flow.
func RecordRouteOutcome(ctx context.Context, outcome string) {
	routeOutcomes.Add(ctx, 1, metric.WithAttributes(attribute.String("outcome", outcome)))
}
