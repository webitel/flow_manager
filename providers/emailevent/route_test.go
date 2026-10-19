package emailevent

import (
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/webitel/flow_manager/model"
)

// Only an error proving that no join happened changes the route.
func TestJoinFailure(t *testing.T) {
	notFound := status.Error(codes.NotFound, joinQueueNotFound)
	precondition := status.Error(codes.FailedPrecondition, joinQueuePrecondition)
	tests := []struct {
		name   string
		from   model.EmailRouteState
		err    error
		state  model.EmailRouteState
		reason string
	}{
		{"unusable queue of the flow", model.EmailRouteJoining, notFound, model.EmailRoutePending, ""},
		{"invalid join of the flow", model.EmailRouteJoining, status.Error(codes.InvalidArgument, "bad"), model.EmailRoutePending, ""},
		{"missing fallback queue", model.EmailRouteResolving, notFound, model.EmailRouteUnrouted, RouteReasonQueueNotFound},
		{"unusable fallback queue", model.EmailRouteResolving, precondition, model.EmailRouteUnrouted, RouteReasonQueuePrecondition},
		{"invalid fallback join", model.EmailRouteResolving, status.Error(codes.InvalidArgument, "bad"), model.EmailRouteFailed, RouteReasonInvalidArgument},
		{"another precondition", model.EmailRouteJoining, status.Error(codes.FailedPrecondition, "other"), "", ""},
		{"unavailable", model.EmailRouteResolving, status.Error(codes.Unavailable, "down"), "", ""},
	}

	for _, tt := range tests {
		res := JoinFailure(tt.from, tt.err)
		switch {
		case tt.state == "" && res != nil:
			t.Errorf("%s: state changes to %+v", tt.name, res)
		case tt.state != "" && (res == nil || res.State != tt.state || res.Reason != tt.reason):
			t.Errorf("%s: %+v, want %s %q", tt.name, res, tt.state, tt.reason)
		}
	}
}
