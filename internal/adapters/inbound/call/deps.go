package call

import (
	"context"

	bscfg "github.com/webitel/flow_manager/internal/bootstrap/config"
	"github.com/webitel/flow_manager/internal/domain/flow"
	"github.com/webitel/flow_manager/internal/domain/notification"
	"github.com/webitel/flow_manager/internal/domain/routing"
	callops "github.com/webitel/flow_manager/internal/runtime/ops/domain/call"
	"github.com/webitel/flow_manager/internal/runtime/runtimekit"
)

// Deps is the full dependency interface for the call channel router.
// It covers direct router calls + all op interfaces passed via ExtraOps.
type Deps interface {
	runtimekit.BootstrapDeps
	AppID() string

	// call routing
	SearchTransferredRouting(domainId int64, schemaId int) (*routing.Routing, error)
	SearchOutboundToDestinationRouting(domainId int64, dest string) (*routing.Routing, error)
	SearchOutboundFromQueueRouting(domainId int64, queueId int) (*routing.Routing, error)
	TransferQueueRouting(domainId int64, queueId int) (*routing.Routing, error)
	TransferAgentRouting(domainId int64, agentId int) (*routing.Routing, error)
	GetRoutingFromDestToGateway(domainId int64, gatewayId int) (*routing.Routing, error)
	SetBlindTransferNumber(domainId int64, callId, destination string) error

	// notifications
	UserNotification(n notification.Notification)

	// settings / logging
	GetSystemSettings(ctx context.Context, domainId int64, name string) (bscfg.SysValue, error)
	StoreCallVariables(id string, vars map[string]string) error
	StoreLog(schemaId int, connId string, log []*flow.StepLog) error

	// ops registered via ExtraOps
	callops.FMDeps
	callops.MediaDeps
	callops.ComplexDeps
}
