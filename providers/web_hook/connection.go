package web_hook

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/webitel/wlog"
	"net/http"
	"regexp"
	"strings"
	"sync"

	"github.com/tidwall/gjson"
	"github.com/webitel/flow_manager/model"
)

var compileVar *regexp.Regexp

func init() {
	compileVar = regexp.MustCompile(`\$\{([\s\S]*?)\}`)
}

type Connection struct {
	id           string
	ctx          context.Context
	stop         context.CancelFunc
	domainId     int64
	schemaId     int
	variables    map[string]string
	responseCode int
	response     http.ResponseWriter
	sync.RWMutex
	log *wlog.Logger
}

func (c *Connection) Log() *wlog.Logger {
	return c.log
}

func (c Connection) Type() model.ConnectionType {
	return model.ConnectionTypeWebHook
}

func (c *Connection) Id() string {
	return c.id
}

func (c *Connection) SchemaId() int {
	return c.schemaId
}

func (c *Connection) NodeId() string {
	return ""
}

func (c *Connection) DomainId() int64 {
	return c.domainId
}

func (c *Connection) Context() context.Context {
	return c.ctx
}

func (c *Connection) Get(key string) (string, bool) {
	c.RLock()
	defer c.RUnlock()

	idx := strings.Index(key, ".")
	if idx > 0 {
		nameRoot := key[0:idx]

		if v, ok := c.variables[nameRoot]; ok {
			return gjson.GetBytes([]byte(v), key[idx+1:]).String(), true
		}
	}
	v, ok := c.variables[key]
	return v, ok
}

func (c *Connection) Set(ctx context.Context, vars model.Variables) (model.Response, *model.AppError) {
	c.Lock()
	defer c.Unlock()

	for k, v := range vars {
		c.variables[k] = fmt.Sprintf("%v", v) // TODO
	}

	return model.CallResponseOK, nil
}

func (c *Connection) ParseText(text string) string {
	text = compileVar.ReplaceAllStringFunc(text, func(varName string) (out string) {
		r := compileVar.FindStringSubmatch(varName)
		if len(r) > 0 {
			out, _ = c.Get(r[1])
		}

		return
	})

	return text
}

func (c *Connection) Close() *model.AppError {
	c.stop()
	return nil
}

func (c *Connection) Variables() map[string]string {
	return c.variables
}

func (c *Connection) SetHeader(k, v string) {
	c.response.Header().Set(k, v)
}

func (c *Connection) WriteBody(data []byte) {
	c.response.Write(data) // TODO
}

func (c *Connection) WriteCode(code int) {
	c.Lock()
	c.responseCode = code
	c.Unlock()
}

func toVariables(in map[string]json.RawMessage) map[string]string {
	vars := make(map[string]string)

	for k, v := range in {
		vars[k] = string(v)
	}

	return vars
}
