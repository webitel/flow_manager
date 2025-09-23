package flow

import (
	"context"
	"github.com/webitel/flow_manager/model"
	"strconv"
)

const (
	variableWbtSockId = "wbt_sock_id"
)

type OpenLinkArgs struct {
	UserId  int64  `json:"userId"`
	Message string `json:"message"`
	Url     string `json:"url"`
}

func (r *router) openLink(_ context.Context, scope *Flow, conn model.Connection, args interface{}) (model.Response, *model.AppError) {
	var argv = OpenLinkArgs{}
	err := scope.Decode(args, &argv)
	if err != nil {
		return nil, err
	}

	if argv.UserId == 0 {
		if uid, ok := conn.Get("user_id"); ok {
			i, _ := strconv.Atoi(uid)
			argv.UserId = int64(i)
		}
	}

	sockId, _ := conn.Get(variableWbtSockId)

	err = r.fm.OpenLink(conn.DomainId(), sockId, argv.UserId, argv.Message, argv.Url)
	if err != nil {
		return nil, err
	}

	return model.CallResponseOK, nil
}
