package builtin

import (
	"context"
	"strconv"

	"github.com/webitel/flow_manager/internal/runtime/ops"
)

// OpenLinkDeps is the narrow interface required by the openLink op.
type OpenLinkDeps interface {
	PushOpenLink(domainId int64, sockId string, userId int64, message, url string) error
}

type openLinkOp struct{ deps OpenLinkDeps }

// OpenLinkOp returns the native openLink op: sends a URL to an agent's
// browser via WebSocket notification.
func OpenLinkOp(deps OpenLinkDeps) ops.Op { return openLinkOp{deps: deps} }

func (openLinkOp) Kind() ops.OpKind { return ops.OpKindSync }

func (o openLinkOp) Execute(ctx context.Context, in ops.OpInput) (ops.OpOutput, error) {
	var userId int64
	switch v := in.Node.Args["userId"].(type) {
	case float64:
		userId = int64(v)
	case string:
		userId, _ = strconv.ParseInt(v, 10, 64)
	}
	if userId == 0 {
		if uid, ok := in.Variables["user_id"]; ok {
			userId, _ = strconv.ParseInt(uid, 10, 64)
		}
	}

	sockId := in.Variables["wbt_sock_id"]

	message, _ := in.Node.Args["message"].(string)
	message = ops.ExpandStr(message, in.Variables, in.GlobalVar)

	url, _ := in.Node.Args["url"].(string)
	url = ops.ExpandStr(url, in.Variables, in.GlobalVar)

	if err := o.deps.PushOpenLink(in.DomainID, sockId, userId, message, url); err != nil {
		return ops.OpOutput{}, err
	}
	return ops.OpOutput{}, nil
}
