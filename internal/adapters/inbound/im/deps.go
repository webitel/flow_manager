package im

import (
	"github.com/webitel/flow_manager/internal/domain/routing"
	imop "github.com/webitel/flow_manager/internal/runtime/ops/domain/im"
	"github.com/webitel/flow_manager/internal/runtime/runtimekit"
	"github.com/webitel/flow_manager/internal/session"
)

// Deps is the narrow interface that the IM router and its ops need.
// *bsruntime.RouterDeps satisfies this interface.
type Deps interface {
	runtimekit.BootstrapDeps
	AppID() string
	CheckpointRepo() session.Repository
	GetChatRouteFromSchemaId(domainId int64, schemaId int32) (*routing.Routing, error)
	imop.QueueDeps
	imop.SendDeps
}
