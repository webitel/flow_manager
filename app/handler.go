package app

import (
	"context"
	"github.com/webitel/flow_manager/model"
)

type Handler interface {
	Request(ctx context.Context, conn model.Connection, req model.ApplicationRequest) (model.Response, *model.AppError)
}
