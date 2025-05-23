// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.3.0
// - protoc             (unknown)
// source: status.proto

package cases

import (
	context "context"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
// Requires gRPC-Go v1.32.0 or later.
const _ = grpc.SupportPackageIsVersion7

const (
	Statuses_ListStatuses_FullMethodName = "/webitel.cases.Statuses/ListStatuses"
	Statuses_CreateStatus_FullMethodName = "/webitel.cases.Statuses/CreateStatus"
	Statuses_UpdateStatus_FullMethodName = "/webitel.cases.Statuses/UpdateStatus"
	Statuses_DeleteStatus_FullMethodName = "/webitel.cases.Statuses/DeleteStatus"
	Statuses_LocateStatus_FullMethodName = "/webitel.cases.Statuses/LocateStatus"
)

// StatusesClient is the client API for Statuses service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type StatusesClient interface {
	// RPC method to list or search statuses
	ListStatuses(ctx context.Context, in *ListStatusRequest, opts ...grpc.CallOption) (*StatusList, error)
	// RPC method to create a new status
	CreateStatus(ctx context.Context, in *CreateStatusRequest, opts ...grpc.CallOption) (*Status, error)
	// RPC method to update an existing status
	UpdateStatus(ctx context.Context, in *UpdateStatusRequest, opts ...grpc.CallOption) (*Status, error)
	// RPC method to delete an existing status
	DeleteStatus(ctx context.Context, in *DeleteStatusRequest, opts ...grpc.CallOption) (*Status, error)
	// RPC method to locate a specific status by ID
	LocateStatus(ctx context.Context, in *LocateStatusRequest, opts ...grpc.CallOption) (*LocateStatusResponse, error)
}

type statusesClient struct {
	cc grpc.ClientConnInterface
}

func NewStatusesClient(cc grpc.ClientConnInterface) StatusesClient {
	return &statusesClient{cc}
}

func (c *statusesClient) ListStatuses(ctx context.Context, in *ListStatusRequest, opts ...grpc.CallOption) (*StatusList, error) {
	out := new(StatusList)
	err := c.cc.Invoke(ctx, Statuses_ListStatuses_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *statusesClient) CreateStatus(ctx context.Context, in *CreateStatusRequest, opts ...grpc.CallOption) (*Status, error) {
	out := new(Status)
	err := c.cc.Invoke(ctx, Statuses_CreateStatus_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *statusesClient) UpdateStatus(ctx context.Context, in *UpdateStatusRequest, opts ...grpc.CallOption) (*Status, error) {
	out := new(Status)
	err := c.cc.Invoke(ctx, Statuses_UpdateStatus_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *statusesClient) DeleteStatus(ctx context.Context, in *DeleteStatusRequest, opts ...grpc.CallOption) (*Status, error) {
	out := new(Status)
	err := c.cc.Invoke(ctx, Statuses_DeleteStatus_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *statusesClient) LocateStatus(ctx context.Context, in *LocateStatusRequest, opts ...grpc.CallOption) (*LocateStatusResponse, error) {
	out := new(LocateStatusResponse)
	err := c.cc.Invoke(ctx, Statuses_LocateStatus_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// StatusesServer is the server API for Statuses service.
// All implementations must embed UnimplementedStatusesServer
// for forward compatibility
type StatusesServer interface {
	// RPC method to list or search statuses
	ListStatuses(context.Context, *ListStatusRequest) (*StatusList, error)
	// RPC method to create a new status
	CreateStatus(context.Context, *CreateStatusRequest) (*Status, error)
	// RPC method to update an existing status
	UpdateStatus(context.Context, *UpdateStatusRequest) (*Status, error)
	// RPC method to delete an existing status
	DeleteStatus(context.Context, *DeleteStatusRequest) (*Status, error)
	// RPC method to locate a specific status by ID
	LocateStatus(context.Context, *LocateStatusRequest) (*LocateStatusResponse, error)
	mustEmbedUnimplementedStatusesServer()
}

// UnimplementedStatusesServer must be embedded to have forward compatible implementations.
type UnimplementedStatusesServer struct {
}

func (UnimplementedStatusesServer) ListStatuses(context.Context, *ListStatusRequest) (*StatusList, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ListStatuses not implemented")
}
func (UnimplementedStatusesServer) CreateStatus(context.Context, *CreateStatusRequest) (*Status, error) {
	return nil, status.Errorf(codes.Unimplemented, "method CreateStatus not implemented")
}
func (UnimplementedStatusesServer) UpdateStatus(context.Context, *UpdateStatusRequest) (*Status, error) {
	return nil, status.Errorf(codes.Unimplemented, "method UpdateStatus not implemented")
}
func (UnimplementedStatusesServer) DeleteStatus(context.Context, *DeleteStatusRequest) (*Status, error) {
	return nil, status.Errorf(codes.Unimplemented, "method DeleteStatus not implemented")
}
func (UnimplementedStatusesServer) LocateStatus(context.Context, *LocateStatusRequest) (*LocateStatusResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method LocateStatus not implemented")
}
func (UnimplementedStatusesServer) mustEmbedUnimplementedStatusesServer() {}

// UnsafeStatusesServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to StatusesServer will
// result in compilation errors.
type UnsafeStatusesServer interface {
	mustEmbedUnimplementedStatusesServer()
}

func RegisterStatusesServer(s grpc.ServiceRegistrar, srv StatusesServer) {
	s.RegisterService(&Statuses_ServiceDesc, srv)
}

func _Statuses_ListStatuses_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ListStatusRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(StatusesServer).ListStatuses(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Statuses_ListStatuses_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(StatusesServer).ListStatuses(ctx, req.(*ListStatusRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Statuses_CreateStatus_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(CreateStatusRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(StatusesServer).CreateStatus(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Statuses_CreateStatus_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(StatusesServer).CreateStatus(ctx, req.(*CreateStatusRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Statuses_UpdateStatus_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(UpdateStatusRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(StatusesServer).UpdateStatus(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Statuses_UpdateStatus_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(StatusesServer).UpdateStatus(ctx, req.(*UpdateStatusRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Statuses_DeleteStatus_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(DeleteStatusRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(StatusesServer).DeleteStatus(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Statuses_DeleteStatus_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(StatusesServer).DeleteStatus(ctx, req.(*DeleteStatusRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Statuses_LocateStatus_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(LocateStatusRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(StatusesServer).LocateStatus(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Statuses_LocateStatus_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(StatusesServer).LocateStatus(ctx, req.(*LocateStatusRequest))
	}
	return interceptor(ctx, in, info, handler)
}

// Statuses_ServiceDesc is the grpc.ServiceDesc for Statuses service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var Statuses_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "webitel.cases.Statuses",
	HandlerType: (*StatusesServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "ListStatuses",
			Handler:    _Statuses_ListStatuses_Handler,
		},
		{
			MethodName: "CreateStatus",
			Handler:    _Statuses_CreateStatus_Handler,
		},
		{
			MethodName: "UpdateStatus",
			Handler:    _Statuses_UpdateStatus_Handler,
		},
		{
			MethodName: "DeleteStatus",
			Handler:    _Statuses_DeleteStatus_Handler,
		},
		{
			MethodName: "LocateStatus",
			Handler:    _Statuses_LocateStatus_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "status.proto",
}
