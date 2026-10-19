package emailevent

import (
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/webitel/flow_manager/model"
)

// Stable ids call_center sends as the status message when the queue cannot take the
// email; the join certainly did not happen.
const (
	joinQueueNotFound     = "store.sql_member.email_join.queue_not_found"
	joinQueuePrecondition = "store.sql_member.email_join.queue_precondition"
)

// Reasons of a route that ended without a queue.
const (
	RouteReasonNoFallbackQueue   = "no_fallback_queue"
	RouteReasonQueueNotFound     = "queue_not_found"
	RouteReasonQueuePrecondition = "queue_precondition"
	RouteReasonInvalidArgument   = "invalid_argument"
	RouteReasonUnexpectedError   = "unexpected_error"
)

// JoinFailure is the next state of a route in from after an error that proves no join happened:
// the Flow intent is dropped, the fallback ends; nil keeps the state, as the join may have happened.
func JoinFailure(from model.EmailRouteState, err error) *model.EmailRouteResult {
	st := status.Convert(err)

	reason := ""
	switch {
	case st.Code() == codes.NotFound && st.Message() == joinQueueNotFound:
		reason = RouteReasonQueueNotFound
	case st.Code() == codes.FailedPrecondition && st.Message() == joinQueuePrecondition:
		reason = RouteReasonQueuePrecondition
	case st.Code() == codes.InvalidArgument:
		reason = RouteReasonInvalidArgument
	default:
		return nil
	}

	res := &model.EmailRouteResult{LastError: err.Error()}
	switch {
	case from == model.EmailRouteJoining:
		res.State = model.EmailRoutePending
	case reason == RouteReasonInvalidArgument:
		res.State, res.Reason = model.EmailRouteFailed, reason
	default:
		res.State, res.Reason = model.EmailRouteUnrouted, reason
	}

	return res
}

// JoinTransient reports an error after which the join may already have happened, so
// only a repeat into the same queue is safe.
func JoinTransient(err error) bool {
	switch status.Code(err) {
	case codes.Unavailable, codes.DeadlineExceeded, codes.Canceled:
		return true
	default:
		return false
	}
}
