package model

import (
	"strings"
	"testing"
	"time"
)

func TestEmailRouteLeaseValidation(t *testing.T) {
	cfg := EmailEventsSettings{
		Prefetch: 20, DeliveryLimit: 20, Workers: 4, Retention: 720 * time.Hour,
		CleanupInterval: time.Hour, CleanupBatch: 1000, RouteTimeout: time.Minute,
		RouteWatcherInterval: time.Second, RouteWatcherBatch: 100,
		RouteLease: 10 * time.Second, RouteRetryMaxInterval: time.Minute, RouteMaxAttempts: 10,
	}
	if err := cfg.Validate(); err == nil || !strings.Contains(err.Error(), "email_route_lease") {
		t.Fatalf("10-second lease must be rejected: %v", err)
	}
	cfg.RouteLease = 15 * time.Second
	if err := cfg.Validate(); err != nil {
		t.Fatalf("15-second lease must be accepted: %v", err)
	}
}
