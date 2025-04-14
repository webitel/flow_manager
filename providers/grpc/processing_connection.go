package grpc

import (
	"context"
	"fmt"
	"github.com/webitel/wlog"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/tidwall/gjson"

	"github.com/webitel/flow_manager/model"
)

type processingConnection struct {
	id         string
	variables  map[string]string
	domainId   int64
	schemaId   int
	parentId   string
	formId     string
	forms      chan model.FormElem
	formAction chan model.FormAction
	finished   chan struct{}

	components map[string]interface{}

	ctx    context.Context
	cancel context.CancelFunc
	sync.RWMutex

	log *wlog.Logger
}

func NewProcessingConnection(domainId int64, schemaId int, vars map[string]string) *processingConnection {
	if vars == nil {
		vars = make(map[string]string)
	}
	var attemptId int
	if tmp, ok := vars["attempt_id"]; ok {
		attemptId, _ = strconv.Atoi(tmp)
	}

	ctx, cancel := context.WithCancel(context.Background())
	id := fmt.Sprintf("%d-%s", attemptId, model.NewId())
	return &processingConnection{
		id:         id,
		domainId:   domainId,
		schemaId:   schemaId,
		variables:  vars,
		ctx:        ctx,
		cancel:     cancel,
		finished:   make(chan struct{}),
		forms:      make(chan model.FormElem, 5),
		components: make(map[string]interface{}),
		log: wlog.GlobalLogger().With(
			wlog.Namespace("context"),
			wlog.String("scope", "processing"),
			wlog.Int("attempt_id", attemptId),
		),
	}
}

func (c *processingConnection) Log() *wlog.Logger {
	return c.log
}

func (c *processingConnection) Context() context.Context {
	return c.ctx
}

func (c *processingConnection) SetComponent(name string, component interface{}) {
	c.Lock()
	c.components[name] = component
	c.Unlock()
}

func (c *processingConnection) GetComponentByName(name string) interface{} {
	c.RLock()
	v, _ := c.components[name]
	c.RUnlock()

	return v
}

func (c *processingConnection) PushForm(ctx context.Context, form model.FormElem) (*model.FormAction, *model.AppError) {
	if c.formAction != nil {
		return nil, model.NewAppError("Processing.PushForm", "processing.form.push.app_err", nil, "Not allow two form", http.StatusInternalServerError)
	}

	c.setActiveFormId(form.Id)

	c.forms <- form

	c.formAction = make(chan model.FormAction)

	select {
	case action, ok := <-c.formAction:
		if ok {
			return &action, nil
		}
	case _, ok := <-ctx.Done():
		if ok {
			return nil, model.NewAppError("Processing.PushForm", "processing.form.push.action", nil, "context cancel", http.StatusInternalServerError)
		}

	}

	return nil, model.NewAppError("Processing.PushForm", "processing.form.push.action", nil, "Form no send action", http.StatusInternalServerError)
}

func (c *processingConnection) FormAction(action model.FormAction) *model.AppError {
	if c.formAction == nil {
		return model.NewAppError("Processing.FillForm", "processing.form.fill.app_err", nil, "Not found active form", http.StatusInternalServerError)
	}
	c.formAction <- action
	close(c.formAction)

	c.formAction = nil
	c.setActiveFormId("")
	return nil
}

func (c *processingConnection) activeFormId() string {
	c.RLock()
	f := c.formId
	c.RUnlock()
	return f
}

func (c *processingConnection) setActiveFormId(id string) {
	c.Lock()
	c.formId = id
	c.Unlock()
}

func (c *processingConnection) ComponentAction(ctx context.Context, formId, componentId string, action string, vars map[string]string, sync bool) *model.AppError {
	if c.formAction == nil {
		return model.NewInternalError("processing.form.app_err", "not found active form")
	}

	if c.activeFormId() != formId {
		return model.NewInternalError("processing.form.app_err", "invalid form id")
	}

	cm := c.GetComponentByName(componentId)
	if cm == nil {
		return model.NewInternalError("processing.form.app_err", fmt.Sprintf("component %s not found", componentId))
	}

	cmp, ok := cm.(model.FormTable)
	if !ok {
		return model.NewInternalError("processing.form.app_err", fmt.Sprintf("component %s does not have outputs", componentId))
	}

	fn, ok := cmp.OutputsFn[action]
	if !ok {
		return model.NewInternalError("processing.form.app_err", fmt.Sprintf("component %s does not have output %s", componentId, action))
	}

	return fn(ctx, sync, model.VariablesFromStringMap(vars))
}

func (c *processingConnection) waitForm(timeSec int) (*model.FormElem, *model.AppError) {
	select {
	case <-time.After(time.Second * time.Duration(timeSec)):
		return nil, model.NewAppError("Processing", "processing.connection.form.timeout", nil, "Timeout", http.StatusBadRequest)
	case <-c.finished:
		return nil, nil
	case <-c.Context().Done():
		return nil, model.NewAppError("Processing", "processing.connection.form.cancel", nil, "Context cancel", http.StatusBadRequest)
	case f, ok := <-c.forms:
		if !ok {
			return nil, model.NewAppError("Processing", "processing.connection.form.close", nil, "Close", http.StatusBadRequest)
		}
		return &f, nil
	}
}

func (c *processingConnection) NodeId() string {
	return "TODO"
}

func (c *processingConnection) ParseText(text string, ops ...model.ParseOption) string {
	return model.ParseText(c, text, ops...)
}

func (c *processingConnection) Id() string {
	return c.id
}

func (c *processingConnection) SchemaId() int {
	return c.schemaId
}

func (c *processingConnection) Close() *model.AppError {
	c.cancel()
	return nil
}

func (c *processingConnection) Finish() {
	close(c.finished)
	c.Close()
}

func (c *processingConnection) DomainId() int64 {
	return c.domainId
}

func (c *processingConnection) Type() model.ConnectionType {
	return model.ConnectionTypeForm
}

func (c *processingConnection) Set(ctx context.Context, vars model.Variables) (model.Response, *model.AppError) {
	c.Lock()
	defer c.Unlock()

	for k, v := range vars {
		c.variables[k] = fmt.Sprintf("%v", v) // TODO
	}

	return model.CallResponseOK, nil
}

func (c *processingConnection) Get(key string) (string, bool) {
	c.RLock()
	defer c.RUnlock()

	idx := strings.Index(key, ".")
	if idx > 0 {
		nameRoot := key[0:idx]

		if v, ok := c.variables[nameRoot]; ok {
			return gjson.GetBytes([]byte(v), key[idx+1:]).String(), true
		}
	}
	if v, ok := c.variables[key]; ok {
		return fmt.Sprintf("%v", v), true
	}

	return "", false
}

func (c *processingConnection) Variables() map[string]string {
	return c.variables
}

//fixme
