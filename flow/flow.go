package flow

import (
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/robertkrimen/otto"

	"github.com/webitel/flow_manager/model"
	"github.com/webitel/wlog"
)

const MAX_GOTO = 1000 //32767

const (
	ApplicationFlagBreakEnabled ApplicationFlag = 1 << iota
	ApplicationFlagAsyncEnabled
)

type ApplicationFlag int

type Flow struct {
	timezone   *time.Location
	handler    Handler
	Connection model.Connection
	name       string
	//Tags       map[string]*Tag
	Functions map[string]model.Applications
	triggers  map[string]model.Applications

	tree LabeledTree

	gotoCounter int16
	cancel      bool
	logs        []*model.StepLog
	vm          *otto.Otto
	sync.RWMutex
}

type Config struct {
	Timezone string
	Name     string
	Handler  Handler
	Schema   model.Applications
	Conn     model.Connection
}

func New(conf Config) *Flow {
	i := &Flow{}
	i.handler = conf.Handler
	i.name = conf.Name
	if conf.Conn != nil {
		i.name += fmt.Sprintf(" [%s]", conf.Conn.Id())
	}
	i.Connection = conf.Conn
	i.Functions = make(map[string]model.Applications)
	i.triggers = make(map[string]model.Applications)
	//i.Tags = make(map[string]*Tag)
	i.logs = make([]*model.StepLog, 0, 1)

	if conf.Timezone != "" {
		i.timezone, _ = time.LoadLocation(conf.Timezone)
	}

	i.tree = NewLabeledTree(conf.Schema)

	return i
}

func (f *Flow) PushSteepLog(name string, s int64) {
	f.Lock()
	f.logs = append(f.logs, &model.StepLog{
		Name:  name,
		Start: s,
		Stop:  model.GetMillis(),
	})
	f.Unlock()
}

func (f *Flow) Logs() []*model.StepLog {
	f.RLock()
	defer f.RUnlock()

	if len(f.logs) > 0 {
		return f.logs
	}

	return nil
}

func (f *Flow) Fork(name string, schema model.Applications) *Flow {
	i := &Flow{}
	i.handler = f.handler
	i.name = name
	i.Connection = f.Connection
	i.Functions = f.Functions
	i.triggers = f.triggers // nil ?
	//i.Tags = make(map[string]*Tag)
	i.timezone = f.timezone

	i.tree = NewLabeledTree(schema)

	return i
}

type Limiter struct {
	count    uint32
	max      uint32
	failover string
}

type Log struct {
	Name string
}

type ApplicationRequest struct {
	args    interface{}
	Flags   ApplicationFlag
	Name    string
	DebugId string
	Tag     string
	limiter *Limiter
	log     *Log
}

func (l *Limiter) MaxCount() bool {
	// todo mutex ?
	if l.count >= l.max {
		return true
	}

	return false
}

func (l *Limiter) AddIteration() {
	l.count++
}

func (a *ApplicationRequest) IsCancel() bool {
	return a.Flags&ApplicationFlagBreakEnabled == ApplicationFlagBreakEnabled
}

func (a *ApplicationRequest) Id() string {
	return a.Name
}

func (a *ApplicationRequest) Args() interface{} {
	return a.args
}

func (i *Flow) Name() string {
	return i.name
}

func (i *Flow) Reset() {
	i.Lock()
	i.cancel = false
	i.Unlock()
}

func (i *Flow) NextRequest() *ApplicationRequest {
	val, ok := i.tree.Next()
	if !ok {
		return nil
	}
	return val.Val
}

func (i *Flow) Goto(tag string) bool {
	if i.gotoCounter > MAX_GOTO {
		wlog.Warn(fmt.Sprintf("call %s max goto count!", i.Connection.Id()))
		return false
	}

	return i.tree.Goto(tag)
}

func (i *Flow) SetCancel() {
	i.Lock()
	defer i.Unlock()
	i.cancel = true
}

func (i *Flow) IsCancel() bool {
	i.RLock()
	defer i.RUnlock()
	return i.cancel
}

func parseReq(m model.ApplicationObject) ApplicationRequest {
	var ok, v bool
	req := ApplicationRequest{}

	for fieldName, fieldValue := range m {
		switch fieldName {
		case "_id":
			if _, ok = fieldValue.(string); ok {
				req.DebugId = fieldValue.(string)
			}
		case "break":
			if v, ok = fieldValue.(bool); ok && v {
				req.Flags |= ApplicationFlagBreakEnabled
			}
		case "async":
			if v, ok = fieldValue.(bool); ok && v {
				req.Flags |= ApplicationFlagAsyncEnabled
			}
		case "tag":
			switch fieldValue.(type) {
			case string:
				req.Tag = fieldValue.(string)
			case int:
				req.Tag = strconv.Itoa(fieldValue.(int))
			}
		case "limit":
			if lim, ok := fieldValue.(map[string]interface{}); ok && lim != nil {
				req.limiter = newLimiter(lim)
			}
		case "trace":
			if l, ok := fieldValue.(map[string]interface{}); ok && l != nil {
				req.log = newLog(l)
			}
		default:
			if req.Name == "" {
				req.Name = fieldName

				if m, ok := fieldValue.(model.ApplicationObject); ok {
					tmp := make(map[string]interface{})
					for argK, argV := range m {
						tmp[argK] = argV
					}
					req.args = tmp
				} else {
					req.args = fieldValue
				}
			}
		}

	}

	if req.Name == "" && req.Flags&ApplicationFlagBreakEnabled == ApplicationFlagBreakEnabled {
		req.Name = "break"
	}
	//FIXME
	//req.setParentNode(root)
	return req
}

func newLimiter(args map[string]interface{}) *Limiter {
	max, _ := args["max"].(float64)
	failover, _ := args["failover"].(string)

	if max > 0 && len(failover) > 0 {
		return &Limiter{
			count:    0,
			max:      uint32(max),
			failover: failover,
		}
	}

	return nil
}

func newLog(args map[string]interface{}) *Log {
	name, _ := args["name"].(string)
	if name != "" {
		return &Log{
			Name: name,
		}
	}

	return nil
}

func ArrInterfaceToArrayApplication(src []interface{}) model.Applications {
	res := make(model.Applications, len(src))
	var ok bool
	var tmp map[string]interface{}

	for k, v := range src {
		if tmp, ok = v.(map[string]interface{}); ok {
			res[k] = tmp
		}
	}
	return res
}
