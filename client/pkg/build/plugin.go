package build

import (
	"os"

	"github.com/containifyci/engine-ci/protos2"
	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/go-plugin"
)

func (p *Plugin) GetBuild() *protos2.BuildArgsResponse {
	return &p.opts
}

type Plugin struct {
	opts protos2.BuildArgsResponse
}

func Serve(opts ...*BuildArgs) {
	logger := hclog.New(&hclog.LoggerOptions{
		Level:           hclog.Error,
		Output:          os.Stderr,
		IncludeLocation: true,
	})

	hclog.SetDefault(logger)

	hclog.Default().SetLevel(hclog.Error)

	logger.Debug("message from plugin", "foo", "bar")

	impl := &Plugin{opts: protos2.BuildArgsResponse{
		Args: opts,
	}}

	plugin.Serve(&plugin.ServeConfig{
		HandshakeConfig: protos2.Handshake,
		Logger:          logger,
		Plugins: map[string]plugin.Plugin{
			"containifyci": &protos2.ContainifyCIGRPCPlugin{Impl: impl},
		},

		// A non-nil value here enables gRPC serving for this plugin...
		GRPCServer: plugin.DefaultGRPCServer,
	})
}
