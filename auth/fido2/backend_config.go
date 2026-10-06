// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package fido2

import (
	"context"
	"net/http"

	"github.com/openbao/openbao/sdk/v2/framework"
	"github.com/openbao/openbao/sdk/v2/logical"
)

type config struct {
	RelyingPartyId      string
	RelyingPartyOrigins []string
}

func (b *backend) loadConfig(ctx context.Context, storage logical.Storage) error {
	b.configLock.Lock()
	defer b.configLock.Unlock()

	entry, err := storage.Get(ctx, "config")
	if err != nil {
		return err
	}

	currentConfig := &config{}
	if entry != nil {
		err = entry.DecodeJSON(currentConfig)
		if err != nil {
			return err
		}
	}

	b.config.Store(currentConfig)

	return nil
}

func (b *backend) pathConfigWrite(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
	relyingPartyId, ok := data.GetOk("relying_party_id")
	if !ok {
		return nil, logical.CodedError(http.StatusBadRequest, "relying_party_id is required")
	}

	relyingPartyOrigins, ok := data.GetOk("relying_party_origins")
	if !ok {
		return nil, logical.CodedError(http.StatusBadRequest, "relying_party_origins is required")
	}

	b.configLock.Lock()
	defer b.configLock.Unlock()

	newConfig := &config{
		RelyingPartyId:      relyingPartyId.(string),
		RelyingPartyOrigins: relyingPartyOrigins.([]string),
	}

	entry, err := logical.StorageEntryJSON("config", newConfig)
	if err != nil {
		return nil, logical.CodedError(http.StatusInternalServerError, "could not marshal config: %v", err)
	}

	err = req.Storage.Put(ctx, entry)
	if err != nil {
		return nil, logical.CodedError(http.StatusInternalServerError, "could not store config: %v", err)
	}

	b.config.Store(newConfig)

	return nil, nil
}

func (b *backend) pathConfigRead(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
	currentConfig := b.config.Load()

	if currentConfig == nil {
		currentConfig = &config{}
	}

	return &logical.Response{
		Data: map[string]any{
			"relying_party_id":      currentConfig.RelyingPartyId,
			"relying_party_origins": currentConfig.RelyingPartyOrigins,
		},
	}, nil
}
