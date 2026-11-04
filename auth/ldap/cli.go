// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package ldap

import (
	"errors"
	"fmt"
	"os"
	"strings"

	pwd "github.com/hashicorp/go-secure-stdlib/password"
	"github.com/openbao/openbao/api/v2"
)

type CLIHandler struct {
	client *api.Client
}

func (h *CLIHandler) ClientWithoutToken() (*api.Client, error) {
	// Read the test client if present
	if h.client != nil {
		return h.client, nil
	}

	config := api.DefaultConfig()

	if err := config.ReadEnvironment(); err != nil {
		return nil, fmt.Errorf("failed to read environment: %w", err)
	}

	// Build the client
	client, err := api.NewClient(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %w", err)
	}

	client.ClearToken()

	// Turn off retries on the CLI
	if api.ReadBaoVariable(api.EnvVaultMaxRetries) == "" {
		client.SetMaxRetries(0)
	}

	return client, nil
}

func (h *CLIHandler) Auth(c *api.Client) (*api.Secret, error) {
	nonInteractive := false
	if v, ok := os.LookupEnv("BAO_CLI_AUTH_NON_INTERACTIVE"); ok && v == "1" {
		nonInteractive = true
	}

	mount := "ldap"
	if v, ok := os.LookupEnv("BAO_CLI_AUTH_MOUNT"); ok && v != "" {
		mount = v
	}

	username := usernameFromEnv()
	if username == "" {
		return nil, errors.New("'username' not supplied and neither 'BAO_CLI_AUTH_USERNAME', 'LOGNAME' nor 'USER' env vars set")
	}
	password := passwordFromEnv()
	if password == "" {
		if nonInteractive {
			return nil, errors.New("'password' not supplied and refusing to pull from stdin")
		}

		fmt.Fprintf(os.Stderr, "Password (will be hidden): ")
		var err error
		password, err = pwd.Read(os.Stdin)
		fmt.Fprintf(os.Stderr, "\n")
		if err != nil {
			return nil, err
		}
	}

	data := map[string]interface{}{
		"password": password,
	}

	path := fmt.Sprintf("auth/%s/login/%s", mount, username)
	secret, err := c.Logical().Write(path, data)
	if err != nil {
		return nil, err
	}
	if secret == nil {
		return nil, errors.New("empty response from credential provider")
	}

	return secret, nil
}

func (h *CLIHandler) Help() string {
	help := `
Usage: BAO_CLI_PLUGIN=1 <plugin-binary> login

  The LDAP auth method allows users to authenticate using LDAP or
  Active Directory. On success, the login response is written to stdout as
  JSON. Prompts and errors are written to stderr, and the exit code is
  non-zero on failure.

  Authenticate as "sally", prompting for the password:

      $ BAO_CLI_PLUGIN=1 BAO_CLI_AUTH_USERNAME=sally <plugin-binary> login
      Password (will be hidden):

  Authenticate as "bob":

      $ BAO_CLI_PLUGIN=1 BAO_CLI_AUTH_USERNAME=bob BAO_CLI_AUTH_PASSWORD=password <plugin-binary> login

Environment:

  BAO_CLI_PLUGIN=1
      Required. Runs the binary in CLI mode instead of as a plugin server.

  BAO_CLI_AUTH_USERNAME=<string>
      LDAP username to use for authentication. If not set, the LOGNAME and
      then USER environment variables are used.

  BAO_CLI_AUTH_PASSWORD=<string>
      LDAP password to use for authentication. If not set, the password is
      prompted for on stdin.

  BAO_CLI_AUTH_MOUNT=<string>
      Path the LDAP auth method is mounted at. Defaults to "ldap".

  The OpenBao server is located through the usual BAO_ADDR, BAO_CACERT and
  related environment variables.
`

	return strings.TrimSpace(help)
}

func usernameFromEnv() string {
	if username := os.Getenv("BAO_CLI_AUTH_USERNAME"); username != "" {
		return username
	}
	if logname := os.Getenv("LOGNAME"); logname != "" {
		return logname
	}
	if user := os.Getenv("USER"); user != "" {
		return user
	}
	return ""
}

func passwordFromEnv() string {
	return api.ReadBaoVariable("BAO_CLI_AUTH_PASSWORD")
}
