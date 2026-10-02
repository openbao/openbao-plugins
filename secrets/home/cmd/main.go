// Command home-engine is the OpenBao external plugin binary for the Home
// secrets engine.
//
// Registration (one-time, after placing the binary in the plugin directory):
//
//	SHA256=$(sha256sum /etc/openbao/plugins/home-engine | cut -d' ' -f1)
//	bao plugin register \
//	    -sha256=$SHA256 \
//	    -command=home-engine \
//	    secret home
//
// Mount:
//
//	bao secrets enable -path=home home
//
// Usage (any token that has an Identity entity):
//
//	bao kv put home/my-secret password=hunter2
//	bao kv get home/my-secret
//	bao kv delete home/my-secret
//	bao kv list home/
package main

import (
	"os"

	backend "github.com/openbao/openbao-plugins/secrets/home"
	"github.com/openbao/openbao/api/v2"
	"github.com/openbao/openbao/sdk/v2/plugin"
)

func main() {
	apiClientMeta := &api.PluginAPIClientMeta{}
	flags := apiClientMeta.FlagSet()
	flags.Parse(os.Args[1:]) //nolint:errcheck

	tlsConfig := apiClientMeta.GetTLSConfig()
	tlsProviderFunc := api.VaultPluginTLSProvider(tlsConfig)

	if err := plugin.ServeMultiplex(&plugin.ServeOpts{
		BackendFactoryFunc: backend.Factory,
		TLSProviderFunc:    tlsProviderFunc,
	}); err != nil {
		os.Stderr.WriteString("error: " + err.Error() + "\n") //nolint:errcheck
		os.Exit(1)
	}
}
