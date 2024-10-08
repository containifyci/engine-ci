// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.4.0
// - protoc             v4.24.4
// source: build.proto

package protos2

import (
	context "context"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
// Requires gRPC-Go v1.62.0 or later.
const _ = grpc.SupportPackageIsVersion8

const (
	ContainifyCIEngine_GetBuild_FullMethodName = "/protos2.ContainifyCIEngine/GetBuild"
)

// ContainifyCIEngineClient is the client API for ContainifyCIEngine service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type ContainifyCIEngineClient interface {
	GetBuild(ctx context.Context, in *Empty, opts ...grpc.CallOption) (*BuildArgsResponse, error)
}

type containifyCIEngineClient struct {
	cc grpc.ClientConnInterface
}

func NewContainifyCIEngineClient(cc grpc.ClientConnInterface) ContainifyCIEngineClient {
	return &containifyCIEngineClient{cc}
}

func (c *containifyCIEngineClient) GetBuild(ctx context.Context, in *Empty, opts ...grpc.CallOption) (*BuildArgsResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(BuildArgsResponse)
	err := c.cc.Invoke(ctx, ContainifyCIEngine_GetBuild_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// ContainifyCIEngineServer is the server API for ContainifyCIEngine service.
// All implementations must embed UnimplementedContainifyCIEngineServer
// for forward compatibility
type ContainifyCIEngineServer interface {
	GetBuild(context.Context, *Empty) (*BuildArgsResponse, error)
	mustEmbedUnimplementedContainifyCIEngineServer()
}

// UnimplementedContainifyCIEngineServer must be embedded to have forward compatible implementations.
type UnimplementedContainifyCIEngineServer struct {
}

func (UnimplementedContainifyCIEngineServer) GetBuild(context.Context, *Empty) (*BuildArgsResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetBuild not implemented")
}
func (UnimplementedContainifyCIEngineServer) mustEmbedUnimplementedContainifyCIEngineServer() {}

// UnsafeContainifyCIEngineServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to ContainifyCIEngineServer will
// result in compilation errors.
type UnsafeContainifyCIEngineServer interface {
	mustEmbedUnimplementedContainifyCIEngineServer()
}

func RegisterContainifyCIEngineServer(s grpc.ServiceRegistrar, srv ContainifyCIEngineServer) {
	s.RegisterService(&ContainifyCIEngine_ServiceDesc, srv)
}

func _ContainifyCIEngine_GetBuild_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(Empty)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ContainifyCIEngineServer).GetBuild(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: ContainifyCIEngine_GetBuild_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ContainifyCIEngineServer).GetBuild(ctx, req.(*Empty))
	}
	return interceptor(ctx, in, info, handler)
}

// ContainifyCIEngine_ServiceDesc is the grpc.ServiceDesc for ContainifyCIEngine service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var ContainifyCIEngine_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "protos2.ContainifyCIEngine",
	HandlerType: (*ContainifyCIEngineServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "GetBuild",
			Handler:    _ContainifyCIEngine_GetBuild_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "build.proto",
}
