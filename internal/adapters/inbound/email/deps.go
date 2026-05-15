package email

import (
	"context"

	bscfg "github.com/webitel/flow_manager/internal/bootstrap/config"
	emailop "github.com/webitel/flow_manager/internal/runtime/ops/domain/email"
	"github.com/webitel/flow_manager/internal/runtime/runtimekit"
)

// Deps is the narrow interface that the email router and its ops need.
// *bsruntime.RouterDeps satisfies this interface.
type Deps interface {
	runtimekit.BootstrapDeps
	AppID() string
	GetSystemSettings(ctx context.Context, domainId int64, name string) (bscfg.SysValue, error)
	MailSetContacts(ctx context.Context, domainId int64, id string, contactIds []int64) error
	emailop.ReplyDeps
}
