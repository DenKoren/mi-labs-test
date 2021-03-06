// Code generated by protoc-gen-go-grpc. DO NOT EDIT.

package v1

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

// ZapuskatorAPIClient is the client API for ZapuskatorAPI service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type ZapuskatorAPIClient interface {
	Calculate(ctx context.Context, in *Calculate_Request, opts ...grpc.CallOption) (*Calculate_Response, error)
	GetContainerInfo(ctx context.Context, in *Container_Request, opts ...grpc.CallOption) (*Container_Response, error)
}

type zapuskatorAPIClient struct {
	cc grpc.ClientConnInterface
}

func NewZapuskatorAPIClient(cc grpc.ClientConnInterface) ZapuskatorAPIClient {
	return &zapuskatorAPIClient{cc}
}

func (c *zapuskatorAPIClient) Calculate(ctx context.Context, in *Calculate_Request, opts ...grpc.CallOption) (*Calculate_Response, error) {
	out := new(Calculate_Response)
	err := c.cc.Invoke(ctx, "/Zapuskator.API.v1.ZapuskatorAPI/Calculate", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *zapuskatorAPIClient) GetContainerInfo(ctx context.Context, in *Container_Request, opts ...grpc.CallOption) (*Container_Response, error) {
	out := new(Container_Response)
	err := c.cc.Invoke(ctx, "/Zapuskator.API.v1.ZapuskatorAPI/GetContainerInfo", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// ZapuskatorAPIServer is the server API for ZapuskatorAPI service.
// All implementations must embed UnimplementedZapuskatorAPIServer
// for forward compatibility
type ZapuskatorAPIServer interface {
	Calculate(context.Context, *Calculate_Request) (*Calculate_Response, error)
	GetContainerInfo(context.Context, *Container_Request) (*Container_Response, error)
	mustEmbedUnimplementedZapuskatorAPIServer()
}

// UnimplementedZapuskatorAPIServer must be embedded to have forward compatible implementations.
type UnimplementedZapuskatorAPIServer struct {
}

func (UnimplementedZapuskatorAPIServer) Calculate(context.Context, *Calculate_Request) (*Calculate_Response, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Calculate not implemented")
}
func (UnimplementedZapuskatorAPIServer) GetContainerInfo(context.Context, *Container_Request) (*Container_Response, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetContainerInfo not implemented")
}
func (UnimplementedZapuskatorAPIServer) mustEmbedUnimplementedZapuskatorAPIServer() {}

// UnsafeZapuskatorAPIServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to ZapuskatorAPIServer will
// result in compilation errors.
type UnsafeZapuskatorAPIServer interface {
	mustEmbedUnimplementedZapuskatorAPIServer()
}

func RegisterZapuskatorAPIServer(s grpc.ServiceRegistrar, srv ZapuskatorAPIServer) {
	s.RegisterService(&ZapuskatorAPI_ServiceDesc, srv)
}

func _ZapuskatorAPI_Calculate_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(Calculate_Request)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ZapuskatorAPIServer).Calculate(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/Zapuskator.API.v1.ZapuskatorAPI/Calculate",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ZapuskatorAPIServer).Calculate(ctx, req.(*Calculate_Request))
	}
	return interceptor(ctx, in, info, handler)
}

func _ZapuskatorAPI_GetContainerInfo_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(Container_Request)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ZapuskatorAPIServer).GetContainerInfo(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/Zapuskator.API.v1.ZapuskatorAPI/GetContainerInfo",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ZapuskatorAPIServer).GetContainerInfo(ctx, req.(*Container_Request))
	}
	return interceptor(ctx, in, info, handler)
}

// ZapuskatorAPI_ServiceDesc is the grpc.ServiceDesc for ZapuskatorAPI service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var ZapuskatorAPI_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "Zapuskator.API.v1.ZapuskatorAPI",
	HandlerType: (*ZapuskatorAPIServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "Calculate",
			Handler:    _ZapuskatorAPI_Calculate_Handler,
		},
		{
			MethodName: "GetContainerInfo",
			Handler:    _ZapuskatorAPI_GetContainerInfo_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "api.v1.proto",
}
