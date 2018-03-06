package plugin

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"sync"

	"github.com/Azure/go-autorest/autorest"

	oidc "github.com/coreos/go-oidc"
	"github.com/hashicorp/errwrap"
	"github.com/hashicorp/go-cleanhttp"
	"github.com/hashicorp/vault/logical"
	"github.com/hashicorp/vault/logical/framework"
)

const (
	metadataURI        = "http://169.254.169.254/metadata/instance"
	metadataAPIVersion = "2017-08-01"

	resourceManagerBaseURI    = "https://management.azure.com/subscriptions"
	resourceManagerAPIVersion = "2017-05-01"

	authorizationBaseURI = "https://login.windows.net"
	issuerBaseURI        = "https://sts.windows.net"
)

func Factory(ctx context.Context, c *logical.BackendConfig) (logical.Backend, error) {
	b := Backend(c)
	if err := b.Setup(ctx, c); err != nil {
		return nil, err
	}
	return b, nil
}

type azureAuthBackend struct {
	*framework.Backend

	l sync.RWMutex

	oidcProvider *oidc.Provider
	oidcVerifier tokenVerifier

	// Oauth2 authorizer for connections to Azure cloud
	authorizer autorest.Authorizer

	httpClient *http.Client
}

func Backend(c *logical.BackendConfig) *azureAuthBackend {
	b := new(azureAuthBackend)

	b.Backend = &framework.Backend{
		AuthRenew:   b.pathLoginRenew,
		BackendType: logical.TypeCredential,
		Invalidate:  b.invalidate,
		Help:        backendHelp,
		PathsSpecial: &logical.Paths{
			Unauthenticated: []string{
				"login",
			},
			SealWrapStorage: []string{
				"config",
			},
		},
		Paths: framework.PathAppend(
			[]*framework.Path{
				pathLogin(b),
				pathConfig(b),
			},
			pathsRole(b),
		),
	}
	b.httpClient = cleanhttp.DefaultClient()

	return b
}

func (b *azureAuthBackend) invalidate(ctx context.Context, key string) {
	switch key {
	case "config":
		b.reset()
	}
}

// Wrapping the IDTokenVerifier to replace in tests
type tokenVerifier interface {
	Verify(context.Context, string) (*oidc.IDToken, error)
}

func (b *azureAuthBackend) getAuthorizers(config *azureConfig) (tokenVerifier, autorest.Authorizer, error) {
	b.l.RLock()
	unlockFunc := b.l.RUnlock
	defer func() { unlockFunc() }()

	if b.oidcVerifier != nil && b.authorizer != nil {
		return b.oidcVerifier, b.authorizer, nil
	}

	// Upgrade lock
	b.l.RUnlock()
	b.l.Lock()
	unlockFunc = b.l.Unlock

	// Check again
	if b.oidcVerifier != nil && b.authorizer != nil {
		return b.oidcVerifier, b.authorizer, nil
	}

	// If tenant id is found in the config, use that.  Otherwise lookup the
	// tenant id from instance metadata.
	tenantID := config.TenantID
	if tenantID == "" {
		var err error
		tenantID, err = b.getTentantID()
		if err != nil {
			return nil, nil, errwrap.Wrapf("unable to determine tenant id: {{err}}", err)
		}
	}

	issuer := fmt.Sprintf("%s/%s/", issuerBaseURI, tenantID)
	provider, err := oidc.NewProvider(context.Background(), issuer)
	if err != nil {
		return nil, nil, err
	}

	authorizer, err := NewAuthorizer(config)
	if err != nil {
		return nil, nil, err
	}

	verifierConfig := &oidc.Config{
		ClientID: config.Resource,
	}
	b.oidcProvider = provider
	b.oidcVerifier = provider.Verifier(verifierConfig)
	b.authorizer = authorizer

	return b.oidcVerifier, b.authorizer, nil
}

func (b *azureAuthBackend) reset() {
	b.l.Lock()
	defer b.l.Unlock()

	b.oidcProvider = nil
	b.oidcVerifier = nil
	b.authorizer = nil
}

type instanceMetadata struct {
	Compute *instanceMetadataCompute `json:"compute"`
}

type instanceMetadataCompute struct {
	SubscriptionID string `json:"subscriptionId"`
}

func (b *azureAuthBackend) getInstanceMetadata() (*instanceMetadata, error) {
	req, _ := http.NewRequest("GET", metadataURI, nil)
	req.Header.Add("metadata", "true")

	q := req.URL.Query()
	q.Add("format", "json")
	q.Add("api-version", metadataAPIVersion)
	req.URL.RawQuery = q.Encode()

	resp, err := b.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, errors.New("invalid status code from instance metadata")
	}

	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, errwrap.Wrapf("unable to read response body: {{err}}", err)
	}

	metadata := new(instanceMetadata)
	if err := json.Unmarshal(respBody, metadata); err != nil {
		return nil, errwrap.Wrapf("unable to unmarshal response: {{err}}", err)
	}
	return metadata, nil
}

// getTenantID retrieves the tenant id using instance metadata. The tenant id
// is used to match to the issuer claim in the signed jwt. Due to the current
// limitation of instance metadata, the tenant id must be derived by making a
// failing call using the subscription id to the resource manager endpoint
// to determine the authorization uri which contains the tenant id. Microsoft
// informs us that this should be available in an upcoming update to the
// instance metadata.
func (b *azureAuthBackend) getTentantID() (string, error) {
	metadata, err := b.getInstanceMetadata()
	if err != nil {
		return "", err
	}
	if metadata == nil || metadata.Compute == nil || metadata.Compute.SubscriptionID == "" {
		return "", errors.New("unable to determine subscription id from instance metadata")
	}

	resourceManagerURI := fmt.Sprintf("%s/%s", resourceManagerBaseURI, metadata.Compute.SubscriptionID)
	req, _ := http.NewRequest("GET", resourceManagerURI, nil)
	q := req.URL.Query()
	q.Add("api-version", resourceManagerAPIVersion)
	req.URL.RawQuery = q.Encode()

	resp, err := b.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	return deriveTenantID(resp)
}

func deriveTenantID(resp *http.Response) (string, error) {
	header := resp.Header.Get("WWW-Authenticate")
	if header == "" {
		return "", fmt.Errorf("WWW-Authenticate header not found in Resource Manager response")
	}

	parts := strings.SplitN(header, " ", 2)
	parts = strings.Split(parts[1], ", ")

	opts := make(map[string]string, len(parts))
	for _, part := range parts {
		vals := strings.SplitN(part, "=", 2)
		key := strings.ToLower(vals[0])
		val := strings.Trim(vals[1], "\",")
		opts[key] = val
	}

	authorizationURI, ok := opts["authorization_uri"]
	if !ok {
		return "", errors.New("could not derive the authorization_uri from response")
	}
	return strings.TrimLeft(authorizationURI, authorizationBaseURI+"/"), nil
}

const backendHelp = `
The Azure backend plugin allows authentication for Azure .
`
