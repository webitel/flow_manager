package model

type CallResponse struct {
	Status string
}

var CallResponseOK = &CallResponse{"SUCCESS"}

type CallRouter interface {
	Router
}

func (r CallResponse) String() string {
	return r.Status
}

type CallDirection string

const (
	CallDirectionInbound  CallDirection = "inbound"
	CallDirectionOutbound               = "outbound"
)

type Call interface {
	Id() string
	Direction() CallDirection
	Destination() string
	UserId() int
	DomainId() int
	InboundGatewayId() int

	RingReady() (Response, *AppError)
	PreAnswer() (Response, *AppError)
	Answer() (Response, *AppError)
	Hangup(cause string) (Response, *AppError)
	HangupNoRoute() (Response, *AppError)
	HangupAppErr() (Response, *AppError)
}
