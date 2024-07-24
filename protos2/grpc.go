// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package protos2

import (
	"context"

	plugin "github.com/hashicorp/go-plugin"
	"google.golang.org/grpc"
)

type ContainifyCIGRPCPlugin struct {
	// GRPCPlugin must still implement the Plugin interface
	plugin.Plugin
	// Concrete implementation, written in Go. This is only used for plugins
	// that are written in Go.
	Impl ContainifyCI
}

func (p *ContainifyCIGRPCPlugin) GRPCServer(broker *plugin.GRPCBroker, s *grpc.Server) error {
	RegisterContainifyCIEngineServer(s, &GRPCServerContainifyCI{Impl: p.Impl, broker: broker})
	return nil
}

func (p *ContainifyCIGRPCPlugin) GRPCClient(ctx context.Context, broker *plugin.GRPCBroker, c *grpc.ClientConn) (interface{}, error) {
	return &ContainifyCIGRPCClient{client: NewContainifyCIEngineClient(c)}, nil
}

// GRPCClient is an implementation of KV that talks over RPC.
type ContainifyCIGRPCClient struct {
	// broker *plugin.GRPCBroker
	client ContainifyCIEngineClient
}

type GRPCServerContainifyCI struct {
	// This is the real implementation
	Impl ContainifyCI

	broker *plugin.GRPCBroker
	UnimplementedContainifyCIEngineServer
	// UnsafeContainifyCIEngineServer
}

func (m *GRPCServerContainifyCI) GetBuild(ctx context.Context, _ *Empty) (*BuildArgsResponse, error) {
	return m.Impl.GetBuild(), nil
}

func (m *ContainifyCIGRPCClient) GetBuild() *BuildArgsResponse {
	args, err := m.client.GetBuild(context.Background(), &Empty{})
	if err != nil {
		panic(err)
	}
	return args
}
