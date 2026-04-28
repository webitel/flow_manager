package ports

// WaitableConnection is the capability a transport connection must expose so
// that the suspendable recvMessage op can register an inbound-message handler
// that fires when the remote end sends a message.
//
// The handler is called from the provider's receive goroutine. It MUST NOT
// block; if work is needed it should be dispatched asynchronously.
//
// OnInboundMessage returns an unregister function. The caller is responsible
// for calling it exactly once when it no longer needs the handler (e.g. after
// the flow resumes or the connection closes).
type WaitableConnection interface {
	OnInboundMessage(handler func(text string)) (unregister func())
}
