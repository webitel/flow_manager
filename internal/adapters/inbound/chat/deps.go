package chat

import (
	"github.com/webitel/flow_manager/internal/domain/routing"
	chatop "github.com/webitel/flow_manager/internal/runtime/ops/domain/chat"
	"github.com/webitel/flow_manager/internal/runtime/runtimekit"
	"github.com/webitel/flow_manager/internal/session"
)

// Deps is the narrow interface that the chat router and its ops need.
// *bsruntime.RouterDeps satisfies this interface.
type Deps interface {
	runtimekit.BootstrapDeps
	AppID() string
	CheckpointRepo() session.Repository
	GetChatRouteFromProfile(domainId, profileId int64) (*routing.Routing, error)
	GetChatRouteFromSchemaId(domainId int64, schemaId int32) (*routing.Routing, error)
	GetChatRouteFromUserId(domainId int64, userId int64) (*routing.Routing, error)
	chatop.ChatDeps
	chatop.SendDeps
	chatop.STTDeps
	chatop.QueueDeps
}
