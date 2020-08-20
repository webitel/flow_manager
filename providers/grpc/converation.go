package grpc

import (
	"context"
	"fmt"
	"github.com/webitel/flow_manager/model"
	"github.com/webitel/wlog"
	"sync"
)

type conversationClient struct {
	id string
}

type message struct {
}

type conversation struct {
	id        int64
	profileId int64
	domainId  int64
	variables map[string]string
	client    *conversationClient
	mx        sync.RWMutex
	ctx       context.Context
	messages  []*message

	confirmation map[string]chan []string

	chat *chatApi

	inbound chan (message)
}

func NewConversation(id, domainId, profileId int64) *conversation {
	return &conversation{
		id:           id,
		profileId:    profileId,
		domainId:     domainId,
		variables:    make(map[string]string),
		client:       nil,
		mx:           sync.RWMutex{},
		ctx:          context.Background(),
		messages:     make([]*message, 5),
		confirmation: make(map[string]chan []string),
		inbound:      make(chan message),
	}
}

func (c conversation) Type() model.ConnectionType {
	return model.ConnectionTypeChat
}

func (c *conversation) Id() string {
	return fmt.Sprintf("%d", c.id) //todo
}

func (c *conversation) NodeId() string {
	return c.client.id
}

func (c *conversation) DomainId() int64 {
	return c.domainId
}

func (c *conversation) Context() context.Context {
	return c.ctx
}

func (c *conversation) Get(name string) (string, bool) {
	v, ok := c.variables[name]
	return v, ok
}

func (c *conversation) Set(ctx context.Context, vars model.Variables) (model.Response, *model.AppError) {
	for k, v := range vars {
		c.variables[k] = fmt.Sprintf("%v", v)
	}
	return model.CallResponseOK, nil
}

func (c *conversation) ParseText(text string) string {
	return text
}

func (c *conversation) Close() *model.AppError {
	return nil // fixme
}

func (c *conversation) Break() *model.AppError {
	c.ctx.Done() //todo
	return nil
}

func (c *conversation) ProfileId() int64 {
	return c.profileId
}

func (c *conversation) Stop(err *model.AppError) {
	if err != nil {
		wlog.Error(fmt.Sprintf("conversation %s stop with error: %s", c.id, err.Error()))
	}

	c.chat.conversations.Remove(c.id)
	wlog.Error("TODO")
}
