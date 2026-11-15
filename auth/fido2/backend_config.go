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
	RelyingPartyId         string
	RelyingPartyOrigins    []string
	RecommendHardwareToken bool
}

func (b *backend) loadConfig(ctx context.Context, storage logical.Storage) (*config, error) {
	entry, err := storage.Get(ctx, configStoragePath)
	if err != nil {
		return nil, err
	}

	c := &config{}
	if entry != nil {
		err = entry.DecodeJSON(c)
		if err != nil {
			return nil, err
		}
	}

	return c, nil

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

	return nil, b.config.Update(func(_ *config) (*config, error) {
		newConfig := &config{
			RelyingPartyId:         relyingPartyId.(string),
			RelyingPartyOrigins:    relyingPartyOrigins.([]string),
			RecommendHardwareToken: data.Get("recommend_hardware_token").(bool),
		}

		entry, err := logical.StorageEntryJSON(configStoragePath, newConfig)
		if err != nil {
			return nil, logical.CodedError(http.StatusInternalServerError, "could not marshal config: %v", err)
		}

		err = req.Storage.Put(ctx, entry)
		if err != nil {
			return nil, logical.CodedError(http.StatusInternalServerError, "could not store config: %v", err)
		}

		return newConfig, nil
	})
}

func (b *backend) pathConfigRead(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
	currentConfig, err := b.config.Get(ctx, req.Storage)
	if err != nil {
		return nil, logical.CodedError(http.StatusInternalServerError, err.Error())
	}

	if currentConfig == nil {
		currentConfig = &config{}
	}

	return &logical.Response{
		Data: map[string]any{
			"relying_party_id":         currentConfig.RelyingPartyId,
			"relying_party_origins":    currentConfig.RelyingPartyOrigins,
			"recommend_hardware_token": currentConfig.RecommendHardwareToken,
		},
	}, nil
}
