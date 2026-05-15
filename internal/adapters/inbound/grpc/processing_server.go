package grpc

import (
	"context"
	"fmt"
	"net/http"

	"github.com/webitel/wlog"

	workflow2 "github.com/webitel/flow_manager/api/gen/workflow"
	"github.com/webitel/flow_manager/internal/domain/flow"
	apperrs "github.com/webitel/flow_manager/internal/infrastructure/errors"
	infraCache "github.com/webitel/flow_manager/internal/infrastructure/cache"
	"github.com/webitel/flow_manager/pkg/processing"
)

var (
	activeProcessingCacheSize = 10000
	waitSecSchemaForm         = 20
)

type processingApi struct {
	connections infraCache.ObjectCache
	sink        chan<- flow.Connection
	nodeName    string
	log         *wlog.Logger
	workflow2.UnsafeFlowProcessingServiceServer
}

func newProcessingApi(sink chan<- flow.Connection, nodeName string) *processingApi {
	return &processingApi{
		sink:        sink,
		nodeName:    nodeName,
		connections: infraCache.NewLru(activeProcessingCacheSize),
		log: wlog.GlobalLogger().With(
			wlog.Namespace("context"),
			wlog.String("scope", "grpc processing"),
		),
	}
}

func (s *processingApi) StartProcessing(ctx context.Context, in *workflow2.StartProcessingRequest) (*workflow2.Form, error) {
	c := NewProcessingConnection(in.DomainId, int(in.SchemaId), in.Variables)
	s.connections.AddWithDefaultExpires(c.id, c)

	c.appId = fmt.Sprintf("workflow-%s", s.nodeName)

	go func() {
		for {
			select {
			case <-c.ctx.Done():
				s.connections.Remove(c.id)
				s.log.With(
					wlog.String("connection_id", c.id),
					wlog.Int("alternative_count", s.connections.Len()),
				).Debug("remove connection")
				return
			}
		}
	}()

	s.sink <- c

	f, err := c.waitForm(waitSecSchemaForm)
	if err != nil {
		c.Close()
		return nil, err
	}

	if f == nil { // END
		return nil, nil
	}

	return &workflow2.Form{
		Id:      c.id,
		AppId:   c.appId,
		Form:    f.ToJson(),
		Timeout: 0,
		Stop:    false,
		Error:   nil,
	}, nil
}

func (s *processingApi) FormAction(ctx context.Context, in *workflow2.FormActionRequest) (*workflow2.Form, error) {
	c, err := s.getProcessingById(in.GetId())
	if err != nil {
		return nil, err
	}

	c.log.With(
		wlog.String("method", in.Action),
		wlog.Any("variables", in.Variables),
	).Debug("receive form action - " + in.Action)

	err = c.FormAction(processing.FormAction{
		Name:   in.Action,
		Fields: flow.VariablesFromStringMap(in.Variables),
	})
	if err != nil {
		return nil, err
	}

	f, err := c.waitForm(waitSecSchemaForm)
	if err != nil {
		c.Close()
		return nil, err
	}

	if f == nil {
		return nil, nil
	}

	return &workflow2.Form{
		Id:      c.id,
		AppId:   c.appId,
		Form:    f.ToJson(),
		Timeout: 0,
		Stop:    false,
		Error:   nil,
	}, nil
}

func (s *processingApi) ComponentAction(ctx context.Context, in *workflow2.ComponentActionRequest) (*workflow2.ComponentActionResponse, error) {
	p, err := s.getProcessingById(in.GetId())
	if err != nil {
		return nil, err
	}

	p.log.With(
		wlog.String("component_id", in.GetComponentId()),
		wlog.String("method", in.Action),
		wlog.Any("variables", in.Variables),
	).Debug("receive component action - " + in.Action)

	err = p.ComponentAction(ctx, in.FormId, in.ComponentId, in.Action, in.Variables, in.Sync)
	if err != nil {
		return nil, err
	}

	return &workflow2.ComponentActionResponse{}, nil
}

func (s *processingApi) CancelProcessing(ctx context.Context, in *workflow2.CancelProcessingRequest) (*workflow2.CancelProcessingResponse, error) {
	c, err := s.getProcessingById(in.GetId())
	if err != nil {
		return nil, err
	}

	if closeErr := c.Close(); closeErr != nil {
		return nil, closeErr
	}
	return &workflow2.CancelProcessingResponse{}, nil
}

func (s *processingApi) getProcessingById(id string) (*processingConnection, error) {
	obj, ok := s.connections.Get(id)
	if !ok {
		return nil, apperrs.New(http.StatusNotFound, "Processing.Get: processing.form.get.not_found: Not found")
	}

	c, ok := obj.(*processingConnection)
	return c, nil
}
