package build

import (
	"os"

	"github.com/containifyci/engine-ci/protos2"
	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/go-plugin"
)

func (p *Plugin) GetBuild() (*protos2.BuildArgsResponse, error) {
	res := &protos2.BuildArgsResponse{
		Args: []*protos2.BuildArgs{},
	}
	for _, arg := range p.builds.Args {
		res.Args = append(res.Args, arg.Args...)
	}
	return res, nil
}

func (p *Plugin) GetBuilds() (*protos2.BuildArgsGroupResponse, error) {
	return &p.builds, nil
}

type Plugin struct {
	builds protos2.BuildArgsGroupResponse
}

// depreacted use Build
func Serve(opts ...*protos2.BuildArgs) {
	Build(opts...)
}

func Build(opts ...*protos2.BuildArgs) {
	args := make([]*protos2.BuildArgsGroup, len(opts))
	for i, opt := range opts {
		args[i] = &protos2.BuildArgsGroup{
			Args: []*protos2.BuildArgs{opt},
		}
	}
	BuildGroups(args...)
}

func BuildAsync(opts ...*protos2.BuildArgs) {
	args := []*protos2.BuildArgsGroup{{
		Args: opts,
	}}
	BuildGroups(args...)
}

func BuildGroups(builds ...*protos2.BuildArgsGroup) {
	logger := hclog.New(&hclog.LoggerOptions{
		Level:           hclog.Error,
		Output:          os.Stderr,
		IncludeLocation: true,
	})

	hclog.SetDefault(logger)

	hclog.Default().SetLevel(hclog.Error)

	logger.Debug("message from plugin", "foo", "bar")

	impl := &Plugin{builds: protos2.BuildArgsGroupResponse{
		Args: builds,
	}}

	plugin.Serve(&plugin.ServeConfig{
		HandshakeConfig: protos2.Handshake,
		Logger:          logger,
		VersionedPlugins: map[int]plugin.PluginSet{
			// Version 2 only uses NetRPC
			1: {
				"containifyci": &protos2.ContainifyCIv1GRPCPlugin{Impl: impl},
			},
			// Version 3 only uses GRPC
			2: {
				"containifyci": &protos2.ContainifyCIv2GRPCPlugin{Impl: impl},
			},
		},
		// A non-nil value here enables gRPC serving for this plugin...
		GRPCServer: plugin.DefaultGRPCServer,
	})
}
