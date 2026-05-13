package call

import (
	"context"

	callops "github.com/webitel/flow_manager/internal/runtime/ops/domain/call"
	"github.com/webitel/flow_manager/internal/runtime/runtimekit"
	"github.com/webitel/flow_manager/model"
)

// Deps is the full dependency interface for the call channel router.
// It covers direct router calls + all op interfaces passed via ExtraOps.
type Deps interface {
	runtimekit.BootstrapDeps
	AppID() string

	// call routing
	SearchTransferredRouting(domainId int64, schemaId int) (*model.Routing, error)
	SearchOutboundToDestinationRouting(domainId int64, dest string) (*model.Routing, error)
	SearchOutboundFromQueueRouting(domainId int64, queueId int) (*model.Routing, error)
	TransferQueueRouting(domainId int64, queueId int) (*model.Routing, error)
	TransferAgentRouting(domainId int64, agentId int) (*model.Routing, error)
	GetRoutingFromDestToGateway(domainId int64, gatewayId int) (*model.Routing, error)
	SetBlindTransferNumber(domainId int64, callId, destination string) error

	// settings / logging
	GetSystemSettings(ctx context.Context, domainId int64, name string) (model.SysValue, error)
	StoreCallVariables(id string, vars map[string]string) error
	StoreLog(schemaId int, connId string, log []*model.StepLog) error

	// ops registered via ExtraOps
	callops.FMDeps
	callops.MediaDeps
	callops.ComplexDeps
}
