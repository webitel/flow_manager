package web_hook

import (
	"fmt"
	"net/http"

	"github.com/webitel/flow_manager/model"
	"github.com/webitel/wlog"
)

type App interface {
	GetHookById(id string) (hook model.WebHook, err *model.AppError)
}

type Context struct {
	app       App
	s         *server
	Err       *model.AppError
	IpAddress string
	Params    Params
}

type Handler struct {
	HandleFunc func(*Context, http.ResponseWriter, *http.Request)
	app        App
	s          *server
}

func (h Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	wlog.Debug(fmt.Sprintf("%v - %v", r.Method, r.URL.Path))
	c := &Context{
		app:       h.app,
		Params:    ParamsFromRequest(r),
		s:         h.s,
		IpAddress: ReadUserIP(r),
	}
	h.HandleFunc(c, w, r)

	if c.Err != nil {
		c.Err.Where = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(c.Err.StatusCode)
		w.Write([]byte(c.Err.ToJson()))
	}
}

func (s *server) ApiHandler(h func(*Context, http.ResponseWriter, *http.Request)) http.Handler {
	return &Handler{
		HandleFunc: h,
		app:        s.app,
		s:          s,
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

func ReadUserIP(r *http.Request) string {
	IPAddress := r.Header.Get("X-Real-Ip")
	if IPAddress == "" {
		IPAddress = r.Header.Get("X-Forwarded-For")
	}
	if IPAddress == "" {
		IPAddress = r.RemoteAddr
	}
	return IPAddress
}
