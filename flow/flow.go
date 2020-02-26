package flow

import (
	"fmt"
	"github.com/webitel/flow_manager/app"
	"github.com/webitel/flow_manager/model"
	"github.com/webitel/wlog"
	"strconv"
	"sync"
)

const MAX_GOTO = 100 //32767

const (
	ApplicationFlagBreakEnabled ApplicationFlag = 1 << iota
	ApplicationFlagAsyncEnabled
)

type ApplicationFlag int

type Tag struct {
	parent *Node
	idx    int
}

type Flow struct {
	fm          *app.FlowManager
	handler     app.Handler
	conn        model.Connection
	name        string
	Tags        map[string]*Tag
	Functions   map[string]*Flow
	triggers    map[string]*Flow
	currentNode *Node
	gotoCounter int16
	cancel      bool
	sync.RWMutex
}

func New(name string, fm *app.FlowManager, handler app.Handler, c model.Applications, conn model.Connection) *Flow {
	i := &Flow{}
	i.fm = fm
	i.handler = handler
	i.name = name
	i.conn = conn
	i.currentNode = NewNode(nil)
	i.Functions = make(map[string]*Flow)
	i.triggers = make(map[string]*Flow)
	i.Tags = make(map[string]*Tag)
	parseFlowArray(i, i.currentNode, c)
	return i
}

type ApplicationRequest struct {
	BaseNode
	args    interface{}
	Flags   ApplicationFlag
	Name    string
	DebugId string
	Tag     string
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

func (i *Flow) SetRoot(root *Node) {
	i.currentNode = root
}

func (i *Flow) NextRequest() *ApplicationRequest {
	var req *ApplicationRequest
	req = i.currentNode.Next()
	if req == nil {
		if newNode := i.GetParentNode(); newNode == nil {
			return nil
		} else {
			return i.NextRequest()
		}
	} else {
		return req
	}
}

func (i *Flow) GetParentNode() *Node {
	parent := i.currentNode.GetParent()
	i.currentNode.setFirst()
	if parent == nil {
		return nil
	}
	i.currentNode = parent
	return i.currentNode
}

func (i *Flow) trySetTag(tag string, parent *Node, idx int) {
	if tag != "" {
		i.Tags[tag] = &Tag{
			parent: parent,
			idx:    idx,
		}
	}
}

func (i *Flow) Goto(tag string) bool {
	if i.gotoCounter > MAX_GOTO {
		wlog.Warn(fmt.Sprintf("call %s max goto count!", i.conn.Id()))
		return false
	}

	if gotoApp, ok := i.Tags[tag]; ok {
		i.currentNode.setFirst()
		i.SetRoot(gotoApp.parent)
		i.currentNode.position = gotoApp.idx
		if i.currentNode.parent != nil {
			i.currentNode.parent.position = i.currentNode.idx + 1
		}
		i.gotoCounter++
		return true
	}
	return false
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

func parseFlowArray(i *Flow, root *Node, apps model.Applications) {
	for _, v := range apps {
		req, err := parseReq(v, root)
		if err != nil {
			wlog.Error(fmt.Sprintf("parse [%v] error: %s", v, err.Error()))
			continue
		}

		switch req.Name {
		case "if":
			req.args = newConditionArgs(i, root, req.args)
			i.trySetTag(req.Tag, root, req.idx)
			root.Add(req)

		case "function":
			if err := i.addFunction(req.args); err != nil {
				wlog.Warn(err.Error())
			}
		case "trigger":
			fmt.Println("trigger")
		case "switch":
			if req.args, err = newSwitchArgs(i, root, req.args); err != nil {
				wlog.Warn(err.Error())
			} else {
				i.trySetTag(req.Tag, root, req.idx)
				root.Add(req)
			}

		case "break":
			fmt.Println("break")

		default:
			if req.Name != "" {
				root.Add(req)
			} else {
				wlog.Warn(fmt.Sprintf("bad application structure %v", v))
			}
		}
	}
}

func parseReq(m model.ApplicationObject, root *Node) (ApplicationRequest, *model.AppError) {
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
	req.setParentNode(root)
	return req, nil
}

func ArrInterfaceToArrayApplication(src []interface{}) model.Applications {
	res := make(model.Applications, len(src))
	var ok bool
	for k, v := range src {
		if _, ok = v.(map[string]interface{}); ok {
			res[k] = v.(map[string]interface{})
		}
	}
	return res
}
