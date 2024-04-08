package grpc

import (
	"context"
	"fmt"
	"net/http"

	"github.com/webitel/flow_manager/model"

	gogrpc "buf.build/gen/go/webitel/workflow/grpc/go/_gogrpc"
	workflow "buf.build/gen/go/webitel/workflow/protocolbuffers/go"
	"github.com/webitel/engine/utils"
	"github.com/webitel/wlog"
)

var (
	activeProcessingCacheSize = 10000
	waitSecSchemaForm         = 10
)

type processingApi struct {
	connections utils.ObjectCache
	*server
	gogrpc.UnsafeFlowProcessingServiceServer
}

func NewProcessingApi(s *server) *processingApi {
	return &processingApi{
		server:      s,
		connections: utils.NewLru(activeProcessingCacheSize),
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
				wlog.Debug(fmt.Sprintf("remove connection %s [%d]", c.id, s.connections.Len()))
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

	wlog.Debug(fmt.Sprintf("[%s] receive action \"%s\" fields: %v", c.id, in.Action, in.Variables))

	err = c.FormAction(model.FormAction{
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
