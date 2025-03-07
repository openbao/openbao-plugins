package main

import (
	"os"

	hclog "github.com/hashicorp/go-hclog"
	"github.com/openbao/openbao-plugin-secrets-oauthapp/v3/pkg/backend"
	"github.com/openbao/openbao/api/v2"
	"github.com/openbao/openbao/sdk/v2/plugin"
)

func main() {
	meta := &api.PluginAPIClientMeta{}

	flags := meta.FlagSet()
	_ = flags.Parse(os.Args[1:])

	err := plugin.ServeMultiplex(&plugin.ServeOpts{
		BackendFactoryFunc: backend.Factory,
		TLSProviderFunc:    api.VaultPluginTLSProvider(meta.GetTLSConfig()),
	})
	if err != nil {
		logger := hclog.New(&hclog.LoggerOptions{})

		logger.Error("plugin shutting down", "error", err)
		os.Exit(1)
	}
}
