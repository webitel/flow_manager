package im

import (
	imop "github.com/webitel/flow_manager/internal/runtime/ops/domain/im"
	"github.com/webitel/flow_manager/internal/runtime/runtimekit"
	"github.com/webitel/flow_manager/internal/session"
	"github.com/webitel/flow_manager/model"
)

// Deps is the narrow interface that the IM router and its ops need.
// *app.FlowManager satisfies this interface.
type Deps interface {
	runtimekit.BootstrapDeps
	AppID() string
	CheckpointRepo() session.Repository
	GetChatRouteFromSchemaId(domainId int64, schemaId int32) (*model.Routing, error)
	imop.QueueDeps
	imop.SendDeps
}
