package plugin

import (
	"context"
	"net/http"
	"sync"

	"github.com/hashicorp/go-cleanhttp"
	"github.com/hashicorp/vault/logical"
	"github.com/hashicorp/vault/logical/framework"
)

const (
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

	client     Client
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

func (b *azureAuthBackend) getClient(config *azureConfig) (Client, error) {
	b.l.RLock()
	unlockFunc := b.l.RUnlock
	defer func() { unlockFunc() }()

	if b.client != nil {
		return b.client, nil
	}

	// Upgrade lock
	b.l.RUnlock()
	b.l.Lock()
	unlockFunc = b.l.Unlock

	if b.client != nil {
		return b.client, nil
	}

	client, err := NewAzureClient(config)
	if err != nil {
		return nil, err
	}

	b.client = client
	return b.client, nil
}

func (b *azureAuthBackend) reset() {
	b.l.Lock()
	defer b.l.Unlock()

	b.client = nil
}

const backendHelp = `
The Azure backend plugin allows authentication for Azure .
`
