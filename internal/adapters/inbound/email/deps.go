package email

import (
	"context"

	emailop "github.com/webitel/flow_manager/internal/runtime/ops/domain/email"
	"github.com/webitel/flow_manager/internal/runtime/runtimekit"
	"github.com/webitel/flow_manager/model"
)

// Deps is the narrow interface that the email router and its ops need.
// *app.FlowManager satisfies this interface.
type Deps interface {
	runtimekit.BootstrapDeps
	AppID() string
	GetSystemSettings(ctx context.Context, domainId int64, name string) (model.SysValue, error)
	MailSetContacts(ctx context.Context, domainId int64, id string, contactIds []int64) error
	emailop.ReplyDeps
}
