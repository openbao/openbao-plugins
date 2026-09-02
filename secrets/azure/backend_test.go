// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package azure

import (
	"context"
	"testing"
	"time"

	"github.com/hashicorp/go-hclog"
	log "github.com/hashicorp/go-hclog"
	"github.com/openbao/openbao/sdk/v2/helper/logging"
	"github.com/openbao/openbao/sdk/v2/logical"
)

const (
	defaultLeaseTTLHr         = 1 * time.Hour
	maxLeaseTTLHr             = 12 * time.Hour
	defaultTestTTL            = 300
	defaultTestMaxTTL         = 3600
	defaultTestExplicitMaxTTL = 7200
)

var (
	testClientID     = "testClientId"
	testClientSecret = "testClientSecret"
)

type testSystemViewEnt struct {
	logical.StaticSystemView
}

func getTestBackendMocked(t *testing.T, initConfig bool) (*azureSecretBackend, logical.Storage) {
	b := backend()
	sysView := testSystemViewEnt{}
	sysView.DefaultLeaseTTLVal = defaultLeaseTTLHr
	sysView.MaxLeaseTTLVal = maxLeaseTTLHr

	config := &logical.BackendConfig{
		Logger:      logging.NewVaultLogger(log.Trace),
		System:      &sysView,
		StorageView: &logical.InmemStorage{},
	}
	err := b.Setup(context.Background(), config)
	if err != nil {
		t.Fatalf("unable to create backend: %v", err)
	}

	b.settings = new(clientSettings)
	mockProvider := newMockProvider()
	b.getProvider = func(context.Context, hclog.Logger, logical.SystemView, *clientSettings) (AzureProvider, error) {
		return mockProvider, nil
	}

	if initConfig {
		cfg := map[string]interface{}{
			"subscription_id":  generateUUID(),
			"tenant_id":        generateUUID(),
			"client_id":        testClientID,
			"client_secret":    testClientSecret,
			"environment":      "AZURECHINACLOUD",
			"ttl":              defaultTestTTL,
			"max_ttl":          defaultTestMaxTTL,
			"explicit_max_ttl": defaultTestExplicitMaxTTL,
		}

		testConfigCreate(t, b, config.StorageView, cfg, false)
	}

	return b, config.StorageView
}

func getTestBackend(t *testing.T) (*azureSecretBackend, logical.Storage) {
	b := backend()

	config := &logical.BackendConfig{
		Logger: logging.NewVaultLogger(log.Trace),
		System: &logical.StaticSystemView{
			DefaultLeaseTTLVal: defaultLeaseTTLHr,
			MaxLeaseTTLVal:     maxLeaseTTLHr,
		},
		StorageView: &logical.InmemStorage{},
	}
	err := b.Setup(context.Background(), config)
	if err != nil {
		t.Fatalf("unable to create backend: %v", err)
	}

	return b, config.StorageView
}

func TestPeriodicFuncNilConfig(t *testing.T) {
	b := backend()

	config := &logical.BackendConfig{
		Logger: logging.NewVaultLogger(log.Trace),
		System: &logical.StaticSystemView{
			DefaultLeaseTTLVal: defaultLeaseTTLHr,
			MaxLeaseTTLVal:     maxLeaseTTLHr,
		},
		StorageView: &logical.InmemStorage{},
	}
	err := b.Setup(context.Background(), config)
	if err != nil {
		t.Fatalf("unable to create backend: %v", err)
	}

	b.settings = new(clientSettings)
	mockProvider := newMockProvider()
	b.getProvider = func(context.Context, hclog.Logger, logical.SystemView, *clientSettings) (AzureProvider, error) {
		return mockProvider, nil
	}

	err = b.periodicFunc(context.Background(), &logical.Request{
		Storage: config.StorageView,
	})
	if err != nil {
		t.Fatalf("periodicFunc error not nil, it should have been: %v", err)
	}
}
