// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package plugin

import (
	"context"
	"os"

	log "github.com/hashicorp/go-hclog"
	"github.com/hashicorp/go-plugin"
	"github.com/openbao/go-kms-wrapping/plugin/v2/pb"
	"github.com/openbao/go-kms-wrapping/v2"
	"google.golang.org/grpc"
)

// ServeOpts configures a KMS plugin server.
type ServeOpts struct {
	WrapperFactoryFunc func() wrapping.Wrapper
	Logger             log.Logger
}

// Serve is used to serve a KMS plugin. This is typically called in the plugin's
// main function.
func Serve(opts *ServeOpts) {
	logger := opts.Logger
	if logger == nil {
		logger = log.New(&log.LoggerOptions{
			Level:      log.Info,
			Output:     os.Stderr,
			JSONFormat: true,
		})
	}

	plugins := map[int]plugin.PluginSet{
		1: {
			"wrapper": &gRPCWrapperPlugin{
				factory: opts.WrapperFactoryFunc,
			},
		},
	}

	plugin.Serve(&plugin.ServeConfig{
		HandshakeConfig:  HandshakeConfig,
		VersionedPlugins: plugins,
		Logger:           logger,
		GRPCServer:       plugin.DefaultGRPCServer,
	})
}

// HandshakeConfig is the handshake config to use with plugins compiled against
// this package.
var HandshakeConfig = plugin.HandshakeConfig{
	MagicCookieKey:   "OPENBAO_KMS_PLUGIN",
	MagicCookieValue: "39704a18-7da7-4bda-9a2d-f7c488d70328",
}

// PluginSets are the versioned plugin sets to use on the client side.
var PluginSets = map[int]plugin.PluginSet{
	1: {
		"wrapper": &gRPCWrapperPlugin{},
	},
}

type gRPCWrapperPlugin struct {
	factory func() wrapping.Wrapper

	// Embedding this will disable the netRPC protocol.
	plugin.NetRPCUnsupportedPlugin
}

func (wp *gRPCWrapperPlugin) GRPCServer(broker *plugin.GRPCBroker, s *grpc.Server) error {
	pb.RegisterWrapperServer(s, &gRPCWrapperServer{
		factory:   wp.factory,
		instances: make(map[string]wrapping.Wrapper),
	})
	return nil
}

func (wp *gRPCWrapperPlugin) GRPCClient(ctx context.Context, broker *plugin.GRPCBroker, c *grpc.ClientConn) (any, error) {
	return &gRPCWrapperClient{
		ctx:    ctx,
		client: pb.NewWrapperClient(c),
	}, nil
}
