// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.2.0
// - protoc             v4.22.2
// source: cluster/cluster.proto

package cluster

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

// ClusterClient is the client API for Cluster service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type ClusterClient interface {
	InitMyCluster(ctx context.Context, in *InitMyClusterRequest, opts ...grpc.CallOption) (*ResponseMessage, error)
	InitOtherCluster(ctx context.Context, in *InitOtherClusterRequest, opts ...grpc.CallOption) (*ResponseMessage, error)
	UpdateMyCluster(ctx context.Context, in *UpdateMyClusterRequest, opts ...grpc.CallOption) (*ResponseMessage, error)
	UpdateOtherCluster(ctx context.Context, in *UpdateOtherClusterRequest, opts ...grpc.CallOption) (*ResponseMessage, error)
	RequestClusterScheduling(ctx context.Context, in *ClusterSchedulingRequest, opts ...grpc.CallOption) (*ClusterSchedulingResponse, error)
}

type clusterClient struct {
	cc grpc.ClientConnInterface
}

func NewClusterClient(cc grpc.ClientConnInterface) ClusterClient {
	return &clusterClient{cc}
}

func (c *clusterClient) InitMyCluster(ctx context.Context, in *InitMyClusterRequest, opts ...grpc.CallOption) (*ResponseMessage, error) {
	out := new(ResponseMessage)
	err := c.cc.Invoke(ctx, "/cluster.Cluster/InitMyCluster", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *clusterClient) InitOtherCluster(ctx context.Context, in *InitOtherClusterRequest, opts ...grpc.CallOption) (*ResponseMessage, error) {
	out := new(ResponseMessage)
	err := c.cc.Invoke(ctx, "/cluster.Cluster/InitOtherCluster", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *clusterClient) UpdateMyCluster(ctx context.Context, in *UpdateMyClusterRequest, opts ...grpc.CallOption) (*ResponseMessage, error) {
	out := new(ResponseMessage)
	err := c.cc.Invoke(ctx, "/cluster.Cluster/UpdateMyCluster", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *clusterClient) UpdateOtherCluster(ctx context.Context, in *UpdateOtherClusterRequest, opts ...grpc.CallOption) (*ResponseMessage, error) {
	out := new(ResponseMessage)
	err := c.cc.Invoke(ctx, "/cluster.Cluster/UpdateOtherCluster", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *clusterClient) RequestClusterScheduling(ctx context.Context, in *ClusterSchedulingRequest, opts ...grpc.CallOption) (*ClusterSchedulingResponse, error) {
	out := new(ClusterSchedulingResponse)
	err := c.cc.Invoke(ctx, "/cluster.Cluster/RequestClusterScheduling", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// ClusterServer is the server API for Cluster service.
// All implementations must embed UnimplementedClusterServer
// for forward compatibility
type ClusterServer interface {
	InitMyCluster(context.Context, *InitMyClusterRequest) (*ResponseMessage, error)
	InitOtherCluster(context.Context, *InitOtherClusterRequest) (*ResponseMessage, error)
	UpdateMyCluster(context.Context, *UpdateMyClusterRequest) (*ResponseMessage, error)
	UpdateOtherCluster(context.Context, *UpdateOtherClusterRequest) (*ResponseMessage, error)
	RequestClusterScheduling(context.Context, *ClusterSchedulingRequest) (*ClusterSchedulingResponse, error)
	mustEmbedUnimplementedClusterServer()
}

// UnimplementedClusterServer must be embedded to have forward compatible implementations.
type UnimplementedClusterServer struct {
}

func (UnimplementedClusterServer) InitMyCluster(context.Context, *InitMyClusterRequest) (*ResponseMessage, error) {
	return nil, status.Errorf(codes.Unimplemented, "method InitMyCluster not implemented")
}
func (UnimplementedClusterServer) InitOtherCluster(context.Context, *InitOtherClusterRequest) (*ResponseMessage, error) {
	return nil, status.Errorf(codes.Unimplemented, "method InitOtherCluster not implemented")
}
func (UnimplementedClusterServer) UpdateMyCluster(context.Context, *UpdateMyClusterRequest) (*ResponseMessage, error) {
	return nil, status.Errorf(codes.Unimplemented, "method UpdateMyCluster not implemented")
}
func (UnimplementedClusterServer) UpdateOtherCluster(context.Context, *UpdateOtherClusterRequest) (*ResponseMessage, error) {
	return nil, status.Errorf(codes.Unimplemented, "method UpdateOtherCluster not implemented")
}
func (UnimplementedClusterServer) RequestClusterScheduling(context.Context, *ClusterSchedulingRequest) (*ClusterSchedulingResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method RequestClusterScheduling not implemented")
}
func (UnimplementedClusterServer) mustEmbedUnimplementedClusterServer() {}

// UnsafeClusterServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to ClusterServer will
// result in compilation errors.
type UnsafeClusterServer interface {
	mustEmbedUnimplementedClusterServer()
}

func RegisterClusterServer(s grpc.ServiceRegistrar, srv ClusterServer) {
	s.RegisterService(&Cluster_ServiceDesc, srv)
}

func _Cluster_InitMyCluster_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(InitMyClusterRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ClusterServer).InitMyCluster(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/cluster.Cluster/InitMyCluster",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ClusterServer).InitMyCluster(ctx, req.(*InitMyClusterRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Cluster_InitOtherCluster_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(InitOtherClusterRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ClusterServer).InitOtherCluster(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/cluster.Cluster/InitOtherCluster",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ClusterServer).InitOtherCluster(ctx, req.(*InitOtherClusterRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Cluster_UpdateMyCluster_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(UpdateMyClusterRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ClusterServer).UpdateMyCluster(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/cluster.Cluster/UpdateMyCluster",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ClusterServer).UpdateMyCluster(ctx, req.(*UpdateMyClusterRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Cluster_UpdateOtherCluster_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(UpdateOtherClusterRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ClusterServer).UpdateOtherCluster(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/cluster.Cluster/UpdateOtherCluster",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ClusterServer).UpdateOtherCluster(ctx, req.(*UpdateOtherClusterRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Cluster_RequestClusterScheduling_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ClusterSchedulingRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ClusterServer).RequestClusterScheduling(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/cluster.Cluster/RequestClusterScheduling",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ClusterServer).RequestClusterScheduling(ctx, req.(*ClusterSchedulingRequest))
	}
	return interceptor(ctx, in, info, handler)
}

// Cluster_ServiceDesc is the grpc.ServiceDesc for Cluster service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var Cluster_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "cluster.Cluster",
	HandlerType: (*ClusterServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "InitMyCluster",
			Handler:    _Cluster_InitMyCluster_Handler,
		},
		{
			MethodName: "InitOtherCluster",
			Handler:    _Cluster_InitOtherCluster_Handler,
		},
		{
			MethodName: "UpdateMyCluster",
			Handler:    _Cluster_UpdateMyCluster_Handler,
		},
		{
			MethodName: "UpdateOtherCluster",
			Handler:    _Cluster_UpdateOtherCluster_Handler,
		},
		{
			MethodName: "RequestClusterScheduling",
			Handler:    _Cluster_RequestClusterScheduling_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "cluster/cluster.proto",
}
