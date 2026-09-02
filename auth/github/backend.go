package github

import (
	"context"
	"fmt"
	"net/url"

	"github.com/google/go-github/github"
	"github.com/hashicorp/go-cleanhttp"
	"github.com/openbao/openbao/sdk/v2/framework"
	"github.com/openbao/openbao/sdk/v2/logical"
	"golang.org/x/oauth2"
)

const operationPrefixGithub = "github"

func Factory(ctx context.Context, conf *logical.BackendConfig) (logical.Backend, error) {
	b := Backend()
	if err := b.Setup(ctx, conf); err != nil {
		return nil, err
	}
	return b, nil
}

// setupPolicyMap creates and configures a PolicyMap for teams or users.
// It sets up the policy map with proper display attributes and operation handlers,
// migrating from the deprecated Callbacks API to the Operations API.
func setupPolicyMap(name, mappingSuffix string) (*framework.PolicyMap, []*framework.Path) {
	policyMap := &framework.PolicyMap{
		PathMap: framework.PathMap{
			Name: name,
		},
		DefaultKey: "default",
	}

	paths := policyMap.Paths()

	// Configure display attributes for the list endpoint
	paths[0].DisplayAttrs = &framework.DisplayAttributes{
		OperationPrefix: operationPrefixGithub,
		OperationSuffix: name,
	}

	// Configure display attributes for the mapping endpoint
	paths[1].DisplayAttrs = &framework.DisplayAttributes{
		OperationPrefix: operationPrefixGithub,
		OperationSuffix: mappingSuffix,
	}

	// Migrate from deprecated Callbacks to Operations API
	paths[0].Operations = map[logical.Operation]framework.OperationHandler{
		logical.ListOperation: &framework.PathOperation{
			Callback: paths[0].Callbacks[logical.ListOperation],
			Summary:  paths[0].HelpSynopsis,
		},
		logical.ReadOperation: &framework.PathOperation{
			Callback: paths[0].Callbacks[logical.ReadOperation],
			Summary:  paths[0].HelpSynopsis,
			DisplayAttrs: &framework.DisplayAttributes{
				OperationVerb:   "list",
				OperationSuffix: name + "2", // The ReadOperation is redundant with the ListOperation
			},
		},
	}

	// Clear deprecated Callbacks after migration
	paths[0].Callbacks = nil

	return policyMap, paths
}

func Backend() *backend {
	var b backend

	// Setup policy maps for teams and users
	teamMap, teamMapPaths := setupPolicyMap("teams", "team-mapping")
	b.TeamMap = teamMap

	userMap, userMapPaths := setupPolicyMap("users", "user-mapping")
	b.UserMap = userMap

	allPaths := append(teamMapPaths, userMapPaths...)
	b.Backend = &framework.Backend{
		Help: backendHelp,

		PathsSpecial: &logical.Paths{
			Unauthenticated: []string{
				"login",
			},
		},

		Paths:       append([]*framework.Path{pathConfig(&b), pathLogin(&b)}, allPaths...),
		AuthRenew:   b.pathLoginRenew,
		BackendType: logical.TypeCredential,
	}

	return &b
}

type backend struct {
	*framework.Backend

	TeamMap *framework.PolicyMap

	UserMap *framework.PolicyMap
}

// Client returns the GitHub client to communicate to GitHub via the
// configured settings.
func (b *backend) Client(token string) (*github.Client, error) {
	tc := cleanhttp.DefaultClient()
	if token != "" {
		ctx := context.WithValue(context.Background(), oauth2.HTTPClient, tc)
		tc = oauth2.NewClient(ctx, &tokenSource{Value: token})
	}

	client := github.NewClient(tc)

	// Set empty upload URL to avoid issues
	emptyUrl, err := url.Parse("")
	if err != nil {
		return nil, fmt.Errorf("failed to parse empty URL: %w", err)
	}
	client.UploadURL = emptyUrl

	return client, nil
}

// tokenSource is an oauth2.TokenSource implementation.
type tokenSource struct {
	Value string
}

func (t *tokenSource) Token() (*oauth2.Token, error) {
	return &oauth2.Token{AccessToken: t.Value}, nil
}

const backendHelp = `
The GitHub credential provider allows authentication via GitHub.

Users provide a personal access token to log in, and the credential
provider verifies they're part of the correct organization and then
maps the user to a set of Vault policies according to the teams they're
part of.

After enabling the credential provider, use the "config" route to
configure it.
`
