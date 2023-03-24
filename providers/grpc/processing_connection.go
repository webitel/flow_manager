package grpc

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/webitel/flow_manager/model"
)

type processingConnection struct {
	id         string
	variables  map[string]string
	domainId   int64
	schemaId   int
	parentId   string
	forms      chan model.FormElem
	formAction chan model.FormAction
	finished   chan struct{}

	components map[string]interface{}

	ctx    context.Context
	cancel context.CancelFunc
	sync.RWMutex
}

func NewProcessingConnection(domainId int64, schemaId int, vars map[string]string) *processingConnection {
	if vars == nil {
		vars = make(map[string]string)
	}

	att, _ := vars["attempt_id"]

	ctx, cancel := context.WithCancel(context.Background())
	return &processingConnection{
		id:         fmt.Sprintf("%s-%s", att, model.NewId()),
		domainId:   domainId,
		schemaId:   schemaId,
		variables:  vars,
		ctx:        ctx,
		cancel:     cancel,
		finished:   make(chan struct{}),
		forms:      make(chan model.FormElem, 5),
		components: make(map[string]interface{}),
	}
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

func (c *processingConnection) PushForm(form model.FormElem) (*model.FormAction, *model.AppError) {
	if c.formAction != nil {
		return nil, model.NewAppError("Processing.PushForm", "processing.form.push.app_err", nil, "Not allow two form", http.StatusInternalServerError)
	}

	c.forms <- form

	c.formAction = make(chan model.FormAction)

	if action, ok := <-c.formAction; ok {
		return &action, nil
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
	return nil
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

func (c *processingConnection) ParseText(text string) string {
	text = compileVar.ReplaceAllStringFunc(text, func(varName string) (out string) {
		r := compileVar.FindStringSubmatch(varName)
		if len(r) > 0 {
			out, _ = c.Get(r[1])
		}

		return
	})

	return text
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

func (c processingConnection) Type() model.ConnectionType {
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

	if v, ok := c.variables[key]; ok {
		return fmt.Sprintf("%v", v), true
	}

	return "", false
}

//fixme
