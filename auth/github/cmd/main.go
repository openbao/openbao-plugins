package main

import (
	"log"
	"os"

	"github.com/openbao/openbao-plugins/auth/github"
	"github.com/openbao/openbao/api/v2"
	"github.com/openbao/openbao/sdk/v2/plugin"
)

func main() {
	apiClientMeta := &api.PluginAPIClientMeta{}
	flags := apiClientMeta.FlagSet()
	if err := flags.Parse(os.Args[1:]); err != nil {
		log.Println("failed to parse flags", "error", err)
		os.Exit(1)
	}

	tlsConfig := apiClientMeta.GetTLSConfig()
	tlsProviderFunc := api.VaultPluginTLSProvider(tlsConfig)

	if err := plugin.ServeMultiplex(&plugin.ServeOpts{
		BackendFactoryFunc: github.Factory,
		// set the TLSProviderFunc so that the plugin maintains backwards
		// compatibility with Vault versions that don’t support plugin AutoMTLS
		TLSProviderFunc: tlsProviderFunc,
	}); err != nil {
		log.Println("plugin shutting down", "error", err)
		os.Exit(1)
	}
}
