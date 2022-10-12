package azureauth

import (
	"context"
	"testing"
	"time"

	log "github.com/hashicorp/go-hclog"
	"github.com/hashicorp/vault/sdk/logical"
)

func getTestBackend(t *testing.T) (*azureAuthBackend, logical.Storage) {
	return getTestBackendWithComputeClient(t, nil, nil, nil)
}

func getTestBackendWithComputeClient(t *testing.T, c computeClientFunc, v vmssClientFunc, m msiClientFunc) (*azureAuthBackend, logical.Storage) {
	t.Helper()
	defaultLeaseTTLVal := time.Hour * 12
	maxLeaseTTLVal := time.Hour * 24
	config := &logical.BackendConfig{
		Logger: log.New(&log.LoggerOptions{Level: log.Trace}),
		System: &logical.StaticSystemView{
			DefaultLeaseTTLVal: defaultLeaseTTLVal,
			MaxLeaseTTLVal:     maxLeaseTTLVal,
		},
		StorageView: &logical.InmemStorage{},
	}
	b := backend()
	err := b.Setup(context.Background(), config)
	if err != nil {
		t.Fatalf("unable to create backend: %v", err)
	}
	b.provider = newMockProvider(c, v, m)
	return b, config.StorageView
}
