package fs

import (
	"context"
	"github.com/webitel/flow_manager/model"
)

const (
	HANGUP_NORMAL_TEMPORARY_FAILURE = "NORMAL_TEMPORARY_FAILURE"
	HANGUP_NO_ROUTE_DESTINATION     = "NO_ROUTE_DESTINATION"
)

func (c *Connection) Answer() (model.Response, *model.AppError) {
	return c.Execute(context.Background(), "answer", "")
}

func (c *Connection) PreAnswer() (model.Response, *model.AppError) {
	return c.Execute(context.Background(), "pre_answer", "")
}

func (c *Connection) RingReady() (model.Response, *model.AppError) {
	return c.Execute(context.Background(), "ring_ready", "")
}

func (c *Connection) Hangup(cause string) (model.Response, *model.AppError) {
	return c.Execute(context.Background(), "hangup", cause)
}

func (c *Connection) HangupNoRoute() (model.Response, *model.AppError) {
	return c.Execute(context.Background(), "hangup", HANGUP_NO_ROUTE_DESTINATION)
}

func (c *Connection) HangupAppErr() (model.Response, *model.AppError) {
	return c.Execute(context.Background(), "hangup", HANGUP_NORMAL_TEMPORARY_FAILURE)
}
