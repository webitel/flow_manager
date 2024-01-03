package web_hook

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/webitel/flow_manager/model"
)

func (s *server) InitApi() {
	s.Router = s.RootRouter.PathPrefix("/").Subrouter()
	hook := s.Router.PathPrefix("/hook").Subrouter()
	hook.Handle("/{id}", s.ApiHandler(startHook)).Methods("POST")
	hook.Handle("/{id}", s.ApiHandler(startHook)).Methods("GET")
}

func startHook(c *Context, w http.ResponseWriter, r *http.Request) {
	var h model.WebHook
	h, c.Err = c.app.GetHookById(c.Params.Id)

	if c.Err != nil {
		return
	}

	if !h.Enabled {
		// todo
	}

	if c.Err = h.Authentication(r); c.Err != nil {
		return
	}

	h.InitOrigin()
	if !h.AllowOrigin(c.IpAddress) {
		c.Err = model.NewAppError("WebHook", "hook.valid.origin", nil, "", http.StatusMethodNotAllowed)
		return
	}

	stopC, stopF := context.WithTimeout(context.TODO(), time.Second*60) // TODO maxTimeout

	conn := &Connection{
		id:        model.NewId(),
		ctx:       stopC,
		stop:      stopF,
		domainId:  h.DomainId,
		schemaId:  h.SchemaId,
		variables: nil,
		RWMutex:   sync.RWMutex{},
		response:  w,
	}

	if r.Method != http.MethodGet {

		body, err := io.ReadAll(r.Body)
		if err != nil {

		}

		err = json.Unmarshal(body, &conn.variables)

		if err != nil {

		}
	}

	if conn.variables == nil {
		conn.variables = make(map[string]string)
		for k, v := range r.URL.Query() {
			conn.variables[k] = v[0]
		}
	}

	c.s.consume <- conn

	<-conn.Context().Done()

	if conn.responseCode > 0 {
		conn.response.WriteHeader(conn.responseCode)
	} else {
		conn.response.WriteHeader(http.StatusOK)
	}

}
