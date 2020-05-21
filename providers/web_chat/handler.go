package web_chat

import (
	"fmt"
	"github.com/webitel/flow_manager/model"
	"github.com/webitel/wlog"
	"net/http"
)

type Context struct {
	app    App
	Err    *model.AppError
	Params Params
}

type Handler struct {
	HandleFunc func(*Context, http.ResponseWriter, *http.Request)
	app        App
}

func (h Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	wlog.Debug(fmt.Sprintf("%v - %v", r.Method, r.URL.Path))
	c := &Context{
		app:    h.app,
		Params: ParamsFromRequest(r),
	}
	h.HandleFunc(c, w, r)

	if c.Err != nil {
		c.Err.Where = r.URL.Path
		w.WriteHeader(c.Err.StatusCode)
		w.Write([]byte(c.Err.ToJson()))
	}
}

func (s *server) ApiHandler(h func(*Context, http.ResponseWriter, *http.Request)) http.Handler {
	return &Handler{
		HandleFunc: h,
		app:        s.app,
	}
}

func NewInvalidParamError(parameter string) *model.AppError {
	err := model.NewAppError("Context", "api.context.invalid_body_param.app_error", map[string]interface{}{"Name": parameter}, "", http.StatusBadRequest)
	return err
}

func (c *Context) SetInvalidParam(parameter string) {
	c.Err = NewInvalidParamError(parameter)
}

func (c *Context) SetInvalidUrlParam(parameter string) {
	c.Err = NewInvalidUrlParamError(parameter)
}

func NewInvalidUrlParamError(parameter string) *model.AppError {
	err := model.NewAppError("Context", "api.context.invalid_url_param.app_error", map[string]interface{}{"Name": parameter}, "", http.StatusBadRequest)
	return err
}

func (c *Context) RequireId() *Context {
	if c.Err != nil {
		return c
	}

	if len(c.Params.Id) == 0 {
		c.SetInvalidUrlParam("id")
	}
	return c
}
