// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

// Package shared contains shared data between the host and plugins.
package protos2

import (
	plugin "github.com/hashicorp/go-plugin"
)

// Handshake is a common handshake that is shared by plugin and host.
var Handshake = plugin.HandshakeConfig{
	ProtocolVersion:  1,
	MagicCookieKey:   "BASIC_PLUGIN",
	MagicCookieValue: "hello",
}

// PluginMap is the map of plugins we can dispense.
// var PluginMap = map[string]plugin.Plugin{
// 	"containifyci": &ContainifyCIGRPCPlugin{},
// }

// PluginMap is the map of plugins we can dispense.
var PluginMap = map[int]plugin.PluginSet{
	1: {
		"containifyci": &ContainifyCIv1GRPCPlugin{},
	},
	2: {
		"containifyci": &ContainifyCIv2GRPCPlugin{},
	},
}

type ContainifyCIv2 interface {
	GetBuilds() *BuildArgsGroupResponse
}

type ContainifyCIv1 interface {
	GetBuild() *BuildArgsResponse
}

var _ plugin.GRPCPlugin = &ContainifyCIv2GRPCPlugin{}
var _ ContainifyCIv2 = &ContainifyCIv2GRPCClient{}

var _ plugin.GRPCPlugin = &ContainifyCIv1GRPCPlugin{}
var _ ContainifyCIv1 = &ContainifyCIv1GRPCClient{}
