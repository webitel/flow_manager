package model

import "github.com/webitel/flow_manager/internal/domain/call"

// Re-export queue/exchange constants (used by infra/mq).
const (
	CallVariableSchemaIds = call.CallVariableSchemaIds

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

// Re-exports for backward compatibility.
type CallResponse = call.CallResponse
type CallDirection = call.CallDirection
type CallEndpoint = call.CallEndpoint
type RTPAggregate = call.RTPAggregate
type RtpStats = call.RtpStats
type CallMediaStats = call.CallMediaStats
type CallAction = call.CallAction
type CallActionData = call.CallActionData
type CallActionDataWithUser = call.CallActionDataWithUser
type QueueInfo = call.QueueInfo
type CallActionInfo = call.CallActionInfo
type CallActionRinging = call.CallActionRinging
type CallActionActive = call.CallActionActive
type CallActionHold = call.CallActionHold
type CallActionHeartbeat = call.CallActionHeartbeat
type CallActionBridge = call.CallActionBridge
type CallActionHangup = call.CallActionHangup
type CallActionSTT = call.CallActionSTT
type CallActionTranscript = call.CallActionTranscript
type CallActionMediaStats = call.CallActionMediaStats
type CallVariables = call.CallVariables
type Call = call.Call
type PlaybackFile = call.PlaybackFile
type HttpFileArgs = call.HttpFileArgs
type TTS = call.TTS
type PlaybackDigits = call.PlaybackDigits
type SpeechMessage = call.SpeechMessage
type GetSpeech = call.GetSpeech
type PlaybackArgs = call.PlaybackArgs
type OutboundCallRequest = call.OutboundCallRequest
type OutboundCallEndpoint = call.OutboundCallEndpoint
type OutboundCallParams = call.OutboundCallParams
type MissedCall = call.MissedCall

// Re-export string constants.
const (
	CallDirectionInbound  = call.CallDirectionInbound
	CallDirectionOutbound = call.CallDirectionOutbound
)

const (
	CallEndpointTypeUser        = call.CallEndpointTypeUser
	CallEndpointTypeGateway     = call.CallEndpointTypeGateway
	CallEndpointTypeDestination = call.CallEndpointTypeDestination
)

const (
	CallActionRingingName    = call.CallActionRingingName
	CallActionActiveName     = call.CallActionActiveName
	CallActionBridgeName     = call.CallActionBridgeName
	CallActionHoldName       = call.CallActionHoldName
	CallActionDtmfName       = call.CallActionDtmfName
	CallActionSTTName        = call.CallActionSTTName
	CallActionHangupName     = call.CallActionHangupName
	CallActionHeartbeatName  = call.CallActionHeartbeatName
	CallActionTranscriptName = call.CallActionTranscriptName
	CallActionStatsName      = call.CallActionStatsName
)

var (
	CallResponseOK    = call.CallResponseOK
	CallResponseError = call.CallResponseError
)
