// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package azureauth

import (
	"context"
	"testing"
	"time"

	"github.com/hashicorp/vault/sdk/helper/automatedrotationutil"
	"github.com/hashicorp/vault/sdk/rotation"

	log "github.com/hashicorp/go-hclog"
	"github.com/hashicorp/vault/sdk/helper/pluginutil"
	"github.com/hashicorp/vault/sdk/logical"
)

type testSystemViewEnt struct {
	logical.StaticSystemView
}

func (d testSystemViewEnt) RegisterRotationJob(_ context.Context, _ *rotation.RotationJobConfigureRequest) (string, error) {
	return "", automatedrotationutil.ErrRotationManagerUnsupported
}

func (d testSystemViewEnt) DeregisterRotationJob(_ context.Context, _ *rotation.RotationJobDeregisterRequest) error {
	return nil
}

func (d testSystemViewEnt) GenerateIdentityToken(_ context.Context, _ *pluginutil.IdentityTokenRequest) (*pluginutil.IdentityTokenResponse, error) {
	return &pluginutil.IdentityTokenResponse{}, nil
}

func getTestBackend(t *testing.T) (*azureAuthBackend, logical.Storage) {
	return getTestBackendWithComputeClient(t, nil, nil, nil, nil, nil)
}

func getTestBackendWithResourceClient(t *testing.T, r resourceClientFunc, p providersClientFunc) (*azureAuthBackend, logical.Storage) {
	t.Helper()

	sysView := testSystemViewEnt{}
	sysView.DefaultLeaseTTLVal = time.Hour * 12
	sysView.MaxLeaseTTLVal = time.Hour * 24

	config := &logical.BackendConfig{
		Logger:      log.New(&log.LoggerOptions{Level: log.Trace}),
		System:      &sysView,
		StorageView: &logical.InmemStorage{},
	}
	b := backend()
	err := b.Setup(context.Background(), config)
	if err != nil {
		t.Fatalf("unable to create backend: %v", err)
	}

	b.provider = &mockProvider{
		resourceClientFunc:  r,
		providersClientFunc: p,
	}
	return b, config.StorageView
}

func getTestBackendWithComputeClient(t *testing.T, c computeClientFunc, v vmssClientFunc, m msiClientFunc, ml msiListFunc, g msGraphClientFunc) (*azureAuthBackend, logical.Storage) {
	t.Helper()
	sysView := testSystemViewEnt{}
	sysView.DefaultLeaseTTLVal = time.Hour * 12
	sysView.MaxLeaseTTLVal = time.Hour * 24

	config := &logical.BackendConfig{
		Logger:      log.New(&log.LoggerOptions{Level: log.Trace}),
		System:      &sysView,
		StorageView: &logical.InmemStorage{},
	}
	b := backend()
	err := b.Setup(context.Background(), config)
	if err != nil {
		t.Fatalf("unable to create backend: %v", err)
	}
	b.provider = newMockProvider(c, v, m, ml, g)
	return b, config.StorageView
}
