package grpc

import (
	"context"
	"github.com/webitel/flow_manager/pkg/processing"
	"net/http"

	"github.com/webitel/flow_manager/model"

	"github.com/webitel/flow_manager/gen/workflow"
	"github.com/webitel/wlog"
)

var (
	activeProcessingCacheSize = 10000
	waitSecSchemaForm         = 20
)

type processingApi struct {
	connections model.ObjectCache
	*server
	workflow.UnsafeFlowProcessingServiceServer
}

func NewProcessingApi(s *server) *processingApi {
	return &processingApi{
		server:      s,
		connections: model.NewLru(activeProcessingCacheSize),
	}
}

func (s *processingApi) StartProcessing(ctx context.Context, in *workflow.StartProcessingRequest) (*workflow.Form, error) {
	c := NewProcessingConnection(in.DomainId, int(in.SchemaId), in.Variables)
	s.connections.AddWithDefaultExpires(c.id, c)

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

	s.server.consume <- c

	f, err := c.waitForm(waitSecSchemaForm)
	if err != nil {
		c.Close()
		return nil, err
	}

	if f == nil { // END
		return nil, nil
	}

	return &workflow.Form{
		Id:      c.id,
		AppId:   s.server.Host(), //TODO
		Form:    f.ToJson(),
		Timeout: 0,
		Stop:    false,
		Error:   nil,
	}, nil
}

func (s *processingApi) FormAction(ctx context.Context, in *workflow.FormActionRequest) (*workflow.Form, error) {
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
		Fields: model.VariablesFromStringMap(in.Variables),
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

	return &workflow.Form{
		Id:      c.id,
		AppId:   s.server.Host(), //TODO
		Form:    f.ToJson(),
		Timeout: 0,
		Stop:    false,
		Error:   nil,
	}, nil
}

func (s *processingApi) ComponentAction(ctx context.Context, in *workflow.ComponentActionRequest) (*workflow.ComponentActionResponse, error) {
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

	return &workflow.ComponentActionResponse{}, nil
}

func (s *processingApi) CancelProcessing(ctx context.Context, in *workflow.CancelProcessingRequest) (*workflow.CancelProcessingResponse, error) {
	c, err := s.getProcessingById(in.GetId())
	if err != nil {
		return nil, err
	}

	if err = c.Close(); err != nil {
		return nil, err
	}
	return &workflow.CancelProcessingResponse{}, nil
}

func (s *processingApi) getProcessingById(id string) (*processingConnection, *model.AppError) {
	obj, ok := s.connections.Get(id)
	if !ok {
		return nil, model.NewAppError("Processing.Get", "processing.form.get.not_found", nil, "Not found", http.StatusNotFound)
	}

	c, ok := obj.(*processingConnection)
	return c, nil
}
