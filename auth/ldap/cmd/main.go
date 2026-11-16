// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	hclog "github.com/hashicorp/go-hclog"
	"github.com/openbao/openbao-plugins/auth/ldap"
	"github.com/openbao/openbao/api/v2"
	"github.com/openbao/openbao/sdk/v2/plugin"
)

func main() {
	if os.Getenv("BAO_CLI_PLUGIN") == "1" {
		if len(os.Args) <= 1 {
			fmt.Fprintln(os.Stderr,
				"This binary is a plugin. When instructed to be used in the CLI, a subcommand should be provided.\n"+
					"If you are unsure what subcommands exist, use help like\n"+
					"./openbao-plugin-auth-example help")
			os.Exit(1)
		}
		handler := ldap.CLIHandler{}
		switch strings.ToLower(os.Args[1]) {
		case "login":
			c, err := handler.ClientWithoutToken()
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
			secret, err := handler.Auth(c)
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
			if err := json.NewEncoder(os.Stdout).Encode(secret); err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
		case "help":
			fmt.Println(handler.Help())
		default:
			fmt.Fprintln(os.Stderr, "Please provide a valid subcommand, use help to see all available subcommands.")
			os.Exit(1)
		}
		return
	}

	apiClientMeta := &api.PluginAPIClientMeta{}
	flags := apiClientMeta.FlagSet()
	flags.Parse(os.Args[1:])

	tlsConfig := apiClientMeta.GetTLSConfig()
	tlsProviderFunc := api.VaultPluginTLSProvider(tlsConfig)

	if err := plugin.ServeMultiplex(&plugin.ServeOpts{
		BackendFactoryFunc: ldap.Factory,
		// set the TLSProviderFunc so that the plugin maintains backwards
		// compatibility with Vault versions that don’t support plugin AutoMTLS
		TLSProviderFunc: tlsProviderFunc,
	}); err != nil {
		logger := hclog.New(&hclog.LoggerOptions{})

		logger.Error("plugin shutting down", "error", err)
		os.Exit(1)
	}
}
