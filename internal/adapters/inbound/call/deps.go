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
	SearchTransferredRouting(domainId int64, schemaId int) (*model.Routing, *model.AppError)
	SearchOutboundToDestinationRouting(domainId int64, dest string) (*model.Routing, *model.AppError)
	SearchOutboundFromQueueRouting(domainId int64, queueId int) (*model.Routing, *model.AppError)
	TransferQueueRouting(domainId int64, queueId int) (*model.Routing, *model.AppError)
	TransferAgentRouting(domainId int64, agentId int) (*model.Routing, *model.AppError)
	GetRoutingFromDestToGateway(domainId int64, gatewayId int) (*model.Routing, *model.AppError)
	SetBlindTransferNumber(domainId int64, callId, destination string) *model.AppError

	// settings / logging
	GetSystemSettings(ctx context.Context, domainId int64, name string) (model.SysValue, *model.AppError)
	StoreCallVariables(id string, vars map[string]string) *model.AppError
	StoreLog(schemaId int, connId string, log []*model.StepLog) *model.AppError

	// ops registered via ExtraOps
	callops.FMDeps
	callops.MediaDeps
	callops.ComplexDeps
}
