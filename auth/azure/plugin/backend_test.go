package plugin

import (
	"context"
	"testing"
	"time"

	"github.com/hashicorp/vault/helper/logformat"
	"github.com/hashicorp/vault/logical"
	log "github.com/mgutz/logxi/v1"
)

func getTestBackend(t *testing.T) (*azureAuthBackend, logical.Storage) {
	return getTestBackendWithComputeClient(t, nil)
}

func getTestBackendWithComputeClient(t *testing.T, f computeClientFunc) (*azureAuthBackend, logical.Storage) {
	defaultLeaseTTLVal := time.Hour * 12
	maxLeaseTTLVal := time.Hour * 24
	config := &logical.BackendConfig{
		Logger: logformat.NewVaultLogger(log.LevelTrace),
		System: &logical.StaticSystemView{
			DefaultLeaseTTLVal: defaultLeaseTTLVal,
			MaxLeaseTTLVal:     maxLeaseTTLVal,
		},
		StorageView: &logical.InmemStorage{},
	}
	b := Backend(config)
	err := b.Setup(context.Background(), config)
	if err != nil {
		t.Fatalf("unable to create backend: %v", err)
	}
	b.provider = newMockProvider(f)
	return b, config.StorageView
}
