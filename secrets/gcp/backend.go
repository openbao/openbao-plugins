// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package gcp

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/hashicorp/errwrap"
	"github.com/hashicorp/go-cleanhttp"
	"github.com/openbao/openbao/sdk/v2/framework"
	"github.com/openbao/openbao/sdk/v2/helper/useragent"
	"github.com/openbao/openbao/sdk/v2/logical"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/iam/v1"
	"google.golang.org/api/option"

	"github.com/openbao/openbao-plugins/secrets/gcp/cache"
	"github.com/openbao/openbao-plugins/secrets/gcp/iamutil"
)

const userAgentPluginName = "secrets-gcp"

const (
	// cacheTime is the duration for which to cache clients and credentials. This
	// must be less than 60 minutes.
	cacheTime = 30 * time.Minute

	// operationPrefixGoogleCloud is used as a prefix for OpenAPI operation id's.
	operationPrefixGoogleCloud = "google-cloud"
)

type backend struct {
	*framework.Backend

	// cache is the internal client/object cache. Callers should never access the
	// cache directly.
	cache *cache.Cache

	// pluginEnv contains Vault version information. It is used in user-agent headers.
	pluginEnv *logical.PluginEnvironment

	resources iamutil.ResourceParser

	rolesetLock             sync.Mutex
	staticAccountLock       sync.Mutex
	impersonatedAccountLock sync.Mutex
}

// Factory returns a new backend as logical.Backend.
func Factory(ctx context.Context, conf *logical.BackendConfig) (logical.Backend, error) {
	b := Backend()
	if err := b.Setup(ctx, conf); err != nil {
		return nil, err
	}
	return b, nil
}

func Backend() *backend {
	b := &backend{
		cache:     cache.New(),
		resources: iamutil.GetEnabledResources(),
	}

	b.Backend = &framework.Backend{
		BackendType: logical.TypeLogical,
		Help:        strings.TrimSpace(backendHelp),
		PathsSpecial: &logical.Paths{
			LocalStorage: []string{
				framework.WALPrefix,
			},
			SealWrapStorage: []string{
				"config",
			},
		},

		Paths: framework.PathAppend(
			[]*framework.Path{
				pathConfig(b),
				pathConfigRotateRoot(b),
				// Roleset
				pathRoleSet(b),
				pathRoleSetList(b),
				pathRoleSetRotateAccount(b),
				pathRoleSetRotateKey(b),
				pathRoleSetSecretAccessToken(b),
				pathRoleSetSecretServiceAccountKey(b),
				deprecatedPathRoleSetSecretAccessToken(b),
				deprecatedPathRoleSetSecretServiceAccountKey(b),
				// Static Account
				pathStaticAccount(b),
				pathStaticAccountList(b),
				pathStaticAccountRotateKey(b),
				pathStaticAccountSecretAccessToken(b),
				pathStaticAccountSecretServiceAccountKey(b),
				// Impersonate
				pathImpersonatedAccount(b),
				pathImpersonatedAccountList(b),
				pathImpersonatedAccountSecretAccessToken(b),
			},
		),
		Secrets: []*framework.Secret{
			secretAccessToken(b),
			secretServiceAccountKey(b),
		},

		InitializeFunc: b.initialize,
		Invalidate:     b.invalidate,

		WALRollback:       b.walRollback,
		WALRollbackMinAge: 5 * time.Minute,
	}

	return b
}

func (b *backend) initialize(ctx context.Context, _ *logical.InitializationRequest) error {
	pluginEnv, err := b.System().PluginEnv(ctx)
	if err != nil {
		return fmt.Errorf("failed to read plugin environment: %w", err)
	}
	b.pluginEnv = pluginEnv

	return nil
}

// IAMAdminClient returns a new IAM client. The client is cached.
func (b *backend) IAMAdminClient(s logical.Storage) (*iam.Service, error) {
	httpClient, err := b.HTTPClient(s)
	if err != nil {
		return nil, errwrap.Wrapf("failed to create IAM HTTP client: {{err}}", err)
	}

	client, err := b.cache.Fetch("iam", cacheTime, func() (interface{}, error) {
		client, err := iam.NewService(context.Background(), option.WithHTTPClient(httpClient))
		if err != nil {
			return nil, errwrap.Wrapf("failed to create IAM client: {{err}}", err)
		}
		client.UserAgent = useragent.PluginString(b.pluginEnv, userAgentPluginName)

		return client, nil
	})
	if err != nil {
		return nil, err
	}

	return client.(*iam.Service), nil
}

// HTTPClient returns a new http.Client that is authenticated using the provided
// credentials. The underlying httpClient is cached among all clients.
func (b *backend) HTTPClient(s logical.Storage) (*http.Client, error) {
	creds, err := b.credentials(s)
	if err != nil {
		return nil, errwrap.Wrapf("failed to create oauth2 http client: {{err}}", err)
	}

	client, err := b.cache.Fetch("HTTPClient", cacheTime, func() (interface{}, error) {
		b.Logger().Debug("creating oauth2 http client")
		ctx := context.WithValue(context.Background(), oauth2.HTTPClient, cleanhttp.DefaultClient())
		return oauth2.NewClient(ctx, creds.TokenSource), nil
	})
	if err != nil {
		return nil, err
	}

	return client.(*http.Client), nil
}

// credentials returns the credentials which were specified in the
// configuration. If no credentials were given during configuration, this uses
// default application credentials. If no default application credentials are
// found, this function returns an error. The credentials are cached in-memory
// for performance.
func (b *backend) credentials(s logical.Storage) (*google.Credentials, error) {
	creds, err := b.cache.Fetch("credentials", cacheTime, func() (interface{}, error) {
		b.Logger().Debug("loading credentials")

		ctx := context.Background()

		cfg, err := getConfig(ctx, s)
		if err != nil {
			return nil, err
		}
		if cfg == nil {
			cfg = &config{}
		}
		// Get creds from the config
		credBytes := []byte(cfg.CredentialsRaw)

		// If credentials were provided, use those. Otherwise fall back to the
		// default application credentials.
		var creds *google.Credentials
		if len(credBytes) > 0 {
			creds, err = google.CredentialsFromJSON(ctx, credBytes, iam.CloudPlatformScope)
			if err != nil {
				return nil, errwrap.Wrapf("failed to parse credentials: {{err}}", err)
			}
		} else {
			creds, err = google.FindDefaultCredentials(ctx, iam.CloudPlatformScope)
			if err != nil {
				return nil, errwrap.Wrapf("failed to get default credentials: {{err}}", err)
			}
		}

		return creds, err
	})
	if err != nil {
		return nil, err
	}
	return creds.(*google.Credentials), nil
}

// ClearCaches deletes all cached clients and credentials.
func (b *backend) ClearCaches() {
	b.cache.Clear()
}

// invalidate resets the plugin. This is called when a key is updated via
// replication.
func (b *backend) invalidate(ctx context.Context, key string) {
	switch key {
	case "config":
		b.ClearCaches()
	}
}

const backendHelp = `
The GCP secrets engine dynamically generates Google Cloud service account keys
and OAuth access tokens based on predefined Cloud IAM policies. This enables
users to gain access to Google Cloud resources without needing to create or
manage a dedicated Google Cloud service account.

After mounting this secrets engine, you can configure the credentials using the
"config/" endpoints. You can generate rolesets using the "rolesets/" endpoints.
`
