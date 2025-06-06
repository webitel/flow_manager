// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.3.0
// - protoc             (unknown)
// source: case_link.proto

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
	CaseLinks_LocateLink_FullMethodName = "/webitel.cases.CaseLinks/LocateLink"
	CaseLinks_CreateLink_FullMethodName = "/webitel.cases.CaseLinks/CreateLink"
	CaseLinks_UpdateLink_FullMethodName = "/webitel.cases.CaseLinks/UpdateLink"
	CaseLinks_DeleteLink_FullMethodName = "/webitel.cases.CaseLinks/DeleteLink"
	CaseLinks_ListLinks_FullMethodName  = "/webitel.cases.CaseLinks/ListLinks"
)

// CaseLinksClient is the client API for CaseLinks service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type CaseLinksClient interface {
	LocateLink(ctx context.Context, in *LocateLinkRequest, opts ...grpc.CallOption) (*CaseLink, error)
	CreateLink(ctx context.Context, in *CreateLinkRequest, opts ...grpc.CallOption) (*CaseLink, error)
	UpdateLink(ctx context.Context, in *UpdateLinkRequest, opts ...grpc.CallOption) (*CaseLink, error)
	DeleteLink(ctx context.Context, in *DeleteLinkRequest, opts ...grpc.CallOption) (*CaseLink, error)
	// With Case
	ListLinks(ctx context.Context, in *ListLinksRequest, opts ...grpc.CallOption) (*CaseLinkList, error)
}

type caseLinksClient struct {
	cc grpc.ClientConnInterface
}

func NewCaseLinksClient(cc grpc.ClientConnInterface) CaseLinksClient {
	return &caseLinksClient{cc}
}

func (c *caseLinksClient) LocateLink(ctx context.Context, in *LocateLinkRequest, opts ...grpc.CallOption) (*CaseLink, error) {
	out := new(CaseLink)
	err := c.cc.Invoke(ctx, CaseLinks_LocateLink_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *caseLinksClient) CreateLink(ctx context.Context, in *CreateLinkRequest, opts ...grpc.CallOption) (*CaseLink, error) {
	out := new(CaseLink)
	err := c.cc.Invoke(ctx, CaseLinks_CreateLink_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *caseLinksClient) UpdateLink(ctx context.Context, in *UpdateLinkRequest, opts ...grpc.CallOption) (*CaseLink, error) {
	out := new(CaseLink)
	err := c.cc.Invoke(ctx, CaseLinks_UpdateLink_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *caseLinksClient) DeleteLink(ctx context.Context, in *DeleteLinkRequest, opts ...grpc.CallOption) (*CaseLink, error) {
	out := new(CaseLink)
	err := c.cc.Invoke(ctx, CaseLinks_DeleteLink_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *caseLinksClient) ListLinks(ctx context.Context, in *ListLinksRequest, opts ...grpc.CallOption) (*CaseLinkList, error) {
	out := new(CaseLinkList)
	err := c.cc.Invoke(ctx, CaseLinks_ListLinks_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// CaseLinksServer is the server API for CaseLinks service.
// All implementations must embed UnimplementedCaseLinksServer
// for forward compatibility
type CaseLinksServer interface {
	LocateLink(context.Context, *LocateLinkRequest) (*CaseLink, error)
	CreateLink(context.Context, *CreateLinkRequest) (*CaseLink, error)
	UpdateLink(context.Context, *UpdateLinkRequest) (*CaseLink, error)
	DeleteLink(context.Context, *DeleteLinkRequest) (*CaseLink, error)
	// With Case
	ListLinks(context.Context, *ListLinksRequest) (*CaseLinkList, error)
	mustEmbedUnimplementedCaseLinksServer()
}

// UnimplementedCaseLinksServer must be embedded to have forward compatible implementations.
type UnimplementedCaseLinksServer struct {
}

func (UnimplementedCaseLinksServer) LocateLink(context.Context, *LocateLinkRequest) (*CaseLink, error) {
	return nil, status.Errorf(codes.Unimplemented, "method LocateLink not implemented")
}
func (UnimplementedCaseLinksServer) CreateLink(context.Context, *CreateLinkRequest) (*CaseLink, error) {
	return nil, status.Errorf(codes.Unimplemented, "method CreateLink not implemented")
}
func (UnimplementedCaseLinksServer) UpdateLink(context.Context, *UpdateLinkRequest) (*CaseLink, error) {
	return nil, status.Errorf(codes.Unimplemented, "method UpdateLink not implemented")
}
func (UnimplementedCaseLinksServer) DeleteLink(context.Context, *DeleteLinkRequest) (*CaseLink, error) {
	return nil, status.Errorf(codes.Unimplemented, "method DeleteLink not implemented")
}
func (UnimplementedCaseLinksServer) ListLinks(context.Context, *ListLinksRequest) (*CaseLinkList, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ListLinks not implemented")
}
func (UnimplementedCaseLinksServer) mustEmbedUnimplementedCaseLinksServer() {}

// UnsafeCaseLinksServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to CaseLinksServer will
// result in compilation errors.
type UnsafeCaseLinksServer interface {
	mustEmbedUnimplementedCaseLinksServer()
}

func RegisterCaseLinksServer(s grpc.ServiceRegistrar, srv CaseLinksServer) {
	s.RegisterService(&CaseLinks_ServiceDesc, srv)
}

func _CaseLinks_LocateLink_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(LocateLinkRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(CaseLinksServer).LocateLink(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: CaseLinks_LocateLink_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(CaseLinksServer).LocateLink(ctx, req.(*LocateLinkRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _CaseLinks_CreateLink_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(CreateLinkRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(CaseLinksServer).CreateLink(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: CaseLinks_CreateLink_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(CaseLinksServer).CreateLink(ctx, req.(*CreateLinkRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _CaseLinks_UpdateLink_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(UpdateLinkRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(CaseLinksServer).UpdateLink(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: CaseLinks_UpdateLink_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(CaseLinksServer).UpdateLink(ctx, req.(*UpdateLinkRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _CaseLinks_DeleteLink_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(DeleteLinkRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(CaseLinksServer).DeleteLink(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: CaseLinks_DeleteLink_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(CaseLinksServer).DeleteLink(ctx, req.(*DeleteLinkRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _CaseLinks_ListLinks_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ListLinksRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(CaseLinksServer).ListLinks(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: CaseLinks_ListLinks_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(CaseLinksServer).ListLinks(ctx, req.(*ListLinksRequest))
	}
	return interceptor(ctx, in, info, handler)
}

// CaseLinks_ServiceDesc is the grpc.ServiceDesc for CaseLinks service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var CaseLinks_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "webitel.cases.CaseLinks",
	HandlerType: (*CaseLinksServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "LocateLink",
			Handler:    _CaseLinks_LocateLink_Handler,
		},
		{
			MethodName: "CreateLink",
			Handler:    _CaseLinks_CreateLink_Handler,
		},
		{
			MethodName: "UpdateLink",
			Handler:    _CaseLinks_UpdateLink_Handler,
		},
		{
			MethodName: "DeleteLink",
			Handler:    _CaseLinks_DeleteLink_Handler,
		},
		{
			MethodName: "ListLinks",
			Handler:    _CaseLinks_ListLinks_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "case_link.proto",
}
