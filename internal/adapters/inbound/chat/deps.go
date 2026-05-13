package chat

import (
	chatop "github.com/webitel/flow_manager/internal/runtime/ops/domain/chat"
	"github.com/webitel/flow_manager/internal/runtime/runtimekit"
	"github.com/webitel/flow_manager/internal/session"
	"github.com/webitel/flow_manager/model"
)

// Deps is the narrow interface that the chat router and its ops need.
// *app.FlowManager satisfies this interface.
type Deps interface {
	runtimekit.BootstrapDeps
	AppID() string
	CheckpointRepo() session.Repository
	GetChatRouteFromProfile(domainId, profileId int64) (*model.Routing, *model.AppError)
	GetChatRouteFromSchemaId(domainId int64, schemaId int32) (*model.Routing, *model.AppError)
	GetChatRouteFromUserId(domainId int64, userId int64) (*model.Routing, *model.AppError)
	chatop.ChatDeps
	chatop.SendDeps
	chatop.STTDeps
	chatop.QueueDeps
}
