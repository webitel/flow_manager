package call

// moved from model/call.go

// Exchange and queue name constants used by the MQ infrastructure layer.
const (
	CallExchange       = "call"
	OpensipsExchange   = "opensips"
	ChatExchange       = "chat"
	FlowExchange       = "flow"
	CallEventQueueName = "workflow-call"
	FlowExecQueueName  = "workflow-exec"
	IMQueueNamePrefix  = "im-delivery.workflow-processor.v1"
	IMExchange         = "im_delivery.broadcast"
	CallCenterExchange = "callcenter"
	CallCenterPrefix   = "workflow-cc"
)

// GranteeHeader is the SIP/gRPC header name that carries the grantee ID.
const GranteeHeader = "wbt_grantee_id"

