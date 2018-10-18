package main

import (
	"os"

	hclog "github.com/hashicorp/go-hclog"
	gcpkms "github.com/hashicorp/vault-plugin-secrets-gcpkms"
	"github.com/hashicorp/vault/helper/pluginutil"
	"github.com/hashicorp/vault/logical/plugin"
)

func main() {
	logger := hclog.New(&hclog.LoggerOptions{})

	defer func() {
		if r := recover(); r != nil {
			logger.Error("plugin paniced", "error", r)
			os.Exit(1)
		}
	}()

	meta := &pluginutil.APIClientMeta{}

	flags := meta.FlagSet()
	flags.Parse(os.Args[1:])

	tlsConfig := meta.GetTLSConfig()
	tlsProviderFunc := pluginutil.VaultPluginTLSProvider(tlsConfig)

	if err := plugin.Serve(&plugin.ServeOpts{
		BackendFactoryFunc: gcpkms.Factory,
		TLSProviderFunc:    tlsProviderFunc,
	}); err != nil {
		logger.Error("plugin shutting down", "error", err)
		os.Exit(1)
	}
}
