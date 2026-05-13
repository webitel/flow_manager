package grpc

import (
	"context"
	"fmt"
	"net/http"

	"github.com/webitel/wlog"

	"github.com/webitel/flow_manager/gen/workflow"
	apperrs "github.com/webitel/flow_manager/internal/infrastructure/errors"
	"github.com/webitel/flow_manager/model"
	"github.com/webitel/flow_manager/pkg/processing"
)

var (
	activeProcessingCacheSize = 10000
	waitSecSchemaForm         = 20
)

type processingApi struct {
	connections model.ObjectCache
	*Server
	workflow.UnsafeFlowProcessingServiceServer
}

func NewProcessingApi(s *Server) *processingApi {
	return &processingApi{
		Server:      s,
		connections: model.NewLru(activeProcessingCacheSize),
	}
}

func (s *processingApi) StartProcessing(ctx context.Context, in *workflow.StartProcessingRequest) (*workflow.Form, error) {
	c := NewProcessingConnection(in.DomainId, int(in.SchemaId), in.Variables)
	s.connections.AddWithDefaultExpires(c.id, c)

	c.appId = fmt.Sprintf("workflow-%s", s.Server.nodeName)

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

	s.Server.consume <- c

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
		AppId:   c.appId,
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
		AppId:   c.appId,
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

	if closeErr := c.Close(); closeErr != nil {
		return nil, closeErr
	}
	return &workflow.CancelProcessingResponse{}, nil
}

func (s *processingApi) getProcessingById(id string) (*processingConnection, error) {
	obj, ok := s.connections.Get(id)
	if !ok {
		return nil, apperrs.New(http.StatusNotFound, "Processing.Get: processing.form.get.not_found: Not found")
	}

	c, ok := obj.(*processingConnection)
	return c, nil
}
