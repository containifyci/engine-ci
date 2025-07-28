// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package protos2

import (
	"context"

	plugin "github.com/hashicorp/go-plugin"
	"google.golang.org/grpc"
)

type ContainifyCIv2GRPCPlugin struct {
	// GRPCPlugin must still implement the Plugin interface
	plugin.Plugin
	// Concrete implementation, written in Go. This is only used for plugins
	// that are written in Go.
	Impl ContainifyCIv2
}

func (p *ContainifyCIv2GRPCPlugin) GRPCServer(broker *plugin.GRPCBroker, s *grpc.Server) error {
	RegisterContainifyCIEngineServer(s, &GRPCServerContainifyCIv2{Impl: p.Impl, broker: broker})
	return nil
}

func (p *ContainifyCIv2GRPCPlugin) GRPCClient(ctx context.Context, broker *plugin.GRPCBroker, c *grpc.ClientConn) (interface{}, error) {
	return &ContainifyCIv2GRPCClient{client: NewContainifyCIEngineClient(c)}, nil
}

type ContainifyCIv1GRPCPlugin struct {
	// GRPCPlugin must still implement the Plugin interface
	plugin.Plugin
	// Concrete implementation, written in Go. This is only used for plugins
	// that are written in Go.
	Impl ContainifyCIv1
}

func (p *ContainifyCIv1GRPCPlugin) GRPCServer(broker *plugin.GRPCBroker, s *grpc.Server) error {
	RegisterContainifyCIEngineServer(s, &GRPCServerContainifyCIv1{Impl: p.Impl, broker: broker})
	return nil
}

func (p *ContainifyCIv1GRPCPlugin) GRPCClient(ctx context.Context, broker *plugin.GRPCBroker, c *grpc.ClientConn) (interface{}, error) {
	return &ContainifyCIv1GRPCClient{client: NewContainifyCIEngineClient(c)}, nil
}

// GRPCClient is an implementation of KV that talks over RPC.
type ContainifyCIv2GRPCClient struct {
	// broker *plugin.GRPCBroker
	client ContainifyCIEngineClient
}

// GRPCClient is an implementation of KV that talks over RPC.
type ContainifyCIv1GRPCClient struct {
	// broker *plugin.GRPCBroker
	client ContainifyCIEngineClient
}

type GRPCServerContainifyCIv1 struct {
	// This is the real implementation
	Impl ContainifyCIv1

	broker *plugin.GRPCBroker
	UnimplementedContainifyCIEngineServer
	// UnsafeContainifyCIEngineServer
}

func (m *GRPCServerContainifyCIv1) GetBuild(ctx context.Context, _ *Empty) (*BuildArgsResponse, error) {
	return m.Impl.GetBuild(), nil
}

func (m *ContainifyCIv1GRPCClient) GetBuild() *BuildArgsResponse {
	args, err := m.client.GetBuild(context.Background(), &Empty{})
	if err != nil {
		panic(err)
	}
	return args
}

type GRPCServerContainifyCIv2 struct {
	// This is the real implementation
	Impl ContainifyCIv2

	broker *plugin.GRPCBroker
	UnimplementedContainifyCIEngineServer
	// UnsafeContainifyCIEngineServer
}

func (m *GRPCServerContainifyCIv2) GetBuilds(ctx context.Context, _ *Empty) (*BuildArgsGroupResponse, error) {
	return m.Impl.GetBuilds(), nil
}

func (m *ContainifyCIv2GRPCClient) GetBuilds() *BuildArgsGroupResponse {
	args, err := m.client.GetBuilds(context.Background(), &Empty{})
	if err != nil {
		panic(err)
	}
	return args
}
