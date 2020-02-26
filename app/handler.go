package app

import "github.com/webitel/flow_manager/model"

type Handler interface {
	Request(conn model.Connection, req model.ApplicationRequest) (model.Response, *model.AppError)
}
