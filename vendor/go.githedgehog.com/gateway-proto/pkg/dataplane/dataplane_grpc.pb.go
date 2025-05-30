// Copyright 2025 Hedgehog
// SPDX-License-Identifier: Apache-2.0

// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.5.1
// - protoc             v6.30.2
// source: proto/dataplane.proto

package dataplane

import (
	context "context"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
// Requires gRPC-Go v1.64.0 or later.
const _ = grpc.SupportPackageIsVersion9

const (
	ConfigService_GetConfig_FullMethodName           = "/config.ConfigService/GetConfig"
	ConfigService_GetConfigGeneration_FullMethodName = "/config.ConfigService/GetConfigGeneration"
	ConfigService_UpdateConfig_FullMethodName        = "/config.ConfigService/UpdateConfig"
)

// ConfigServiceClient is the client API for ConfigService service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type ConfigServiceClient interface {
	GetConfig(ctx context.Context, in *GetConfigRequest, opts ...grpc.CallOption) (*GatewayConfig, error)
	GetConfigGeneration(ctx context.Context, in *GetConfigGenerationRequest, opts ...grpc.CallOption) (*GetConfigGenerationResponse, error)
	UpdateConfig(ctx context.Context, in *UpdateConfigRequest, opts ...grpc.CallOption) (*UpdateConfigResponse, error)
}

type configServiceClient struct {
	cc grpc.ClientConnInterface
}

func NewConfigServiceClient(cc grpc.ClientConnInterface) ConfigServiceClient {
	return &configServiceClient{cc}
}

func (c *configServiceClient) GetConfig(ctx context.Context, in *GetConfigRequest, opts ...grpc.CallOption) (*GatewayConfig, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(GatewayConfig)
	err := c.cc.Invoke(ctx, ConfigService_GetConfig_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *configServiceClient) GetConfigGeneration(ctx context.Context, in *GetConfigGenerationRequest, opts ...grpc.CallOption) (*GetConfigGenerationResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(GetConfigGenerationResponse)
	err := c.cc.Invoke(ctx, ConfigService_GetConfigGeneration_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *configServiceClient) UpdateConfig(ctx context.Context, in *UpdateConfigRequest, opts ...grpc.CallOption) (*UpdateConfigResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(UpdateConfigResponse)
	err := c.cc.Invoke(ctx, ConfigService_UpdateConfig_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// ConfigServiceServer is the server API for ConfigService service.
// All implementations must embed UnimplementedConfigServiceServer
// for forward compatibility.
type ConfigServiceServer interface {
	GetConfig(context.Context, *GetConfigRequest) (*GatewayConfig, error)
	GetConfigGeneration(context.Context, *GetConfigGenerationRequest) (*GetConfigGenerationResponse, error)
	UpdateConfig(context.Context, *UpdateConfigRequest) (*UpdateConfigResponse, error)
	mustEmbedUnimplementedConfigServiceServer()
}

// UnimplementedConfigServiceServer must be embedded to have
// forward compatible implementations.
//
// NOTE: this should be embedded by value instead of pointer to avoid a nil
// pointer dereference when methods are called.
type UnimplementedConfigServiceServer struct{}

func (UnimplementedConfigServiceServer) GetConfig(context.Context, *GetConfigRequest) (*GatewayConfig, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetConfig not implemented")
}
func (UnimplementedConfigServiceServer) GetConfigGeneration(context.Context, *GetConfigGenerationRequest) (*GetConfigGenerationResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetConfigGeneration not implemented")
}
func (UnimplementedConfigServiceServer) UpdateConfig(context.Context, *UpdateConfigRequest) (*UpdateConfigResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method UpdateConfig not implemented")
}
func (UnimplementedConfigServiceServer) mustEmbedUnimplementedConfigServiceServer() {}
func (UnimplementedConfigServiceServer) testEmbeddedByValue()                       {}

// UnsafeConfigServiceServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to ConfigServiceServer will
// result in compilation errors.
type UnsafeConfigServiceServer interface {
	mustEmbedUnimplementedConfigServiceServer()
}

func RegisterConfigServiceServer(s grpc.ServiceRegistrar, srv ConfigServiceServer) {
	// If the following call pancis, it indicates UnimplementedConfigServiceServer was
	// embedded by pointer and is nil.  This will cause panics if an
	// unimplemented method is ever invoked, so we test this at initialization
	// time to prevent it from happening at runtime later due to I/O.
	if t, ok := srv.(interface{ testEmbeddedByValue() }); ok {
		t.testEmbeddedByValue()
	}
	s.RegisterService(&ConfigService_ServiceDesc, srv)
}

func _ConfigService_GetConfig_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(GetConfigRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ConfigServiceServer).GetConfig(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: ConfigService_GetConfig_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ConfigServiceServer).GetConfig(ctx, req.(*GetConfigRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _ConfigService_GetConfigGeneration_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(GetConfigGenerationRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ConfigServiceServer).GetConfigGeneration(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: ConfigService_GetConfigGeneration_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ConfigServiceServer).GetConfigGeneration(ctx, req.(*GetConfigGenerationRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _ConfigService_UpdateConfig_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(UpdateConfigRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ConfigServiceServer).UpdateConfig(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: ConfigService_UpdateConfig_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ConfigServiceServer).UpdateConfig(ctx, req.(*UpdateConfigRequest))
	}
	return interceptor(ctx, in, info, handler)
}

// ConfigService_ServiceDesc is the grpc.ServiceDesc for ConfigService service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var ConfigService_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "config.ConfigService",
	HandlerType: (*ConfigServiceServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "GetConfig",
			Handler:    _ConfigService_GetConfig_Handler,
		},
		{
			MethodName: "GetConfigGeneration",
			Handler:    _ConfigService_GetConfigGeneration_Handler,
		},
		{
			MethodName: "UpdateConfig",
			Handler:    _ConfigService_UpdateConfig_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "proto/dataplane.proto",
}
