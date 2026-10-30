// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package fido2

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"path"

	"github.com/openbao/openbao/sdk/v2/framework"
	"github.com/openbao/openbao/sdk/v2/helper/roottoken"
	"github.com/openbao/openbao/sdk/v2/logical"

	"github.com/go-webauthn/webauthn/protocol"
)

func findCredentialsForAlias(ctx context.Context, s logical.Storage, aliasName string) ([]protocol.CredentialDescriptor, error) {
	var credentials []protocol.CredentialDescriptor
	err := logical.WithTransaction(ctx, s, func(s logical.Storage) error {
		basePath := escapedPathJoin("v1", "credentials", aliasName) + "/"
		list, err := s.List(ctx, basePath)
		if err != nil {
			return err
		}

		credentials = make([]protocol.CredentialDescriptor, 0, len(list))

		for _, id := range list {
			entryPath := path.Join(basePath, id)
			entry, err := s.Get(ctx, entryPath)
			if err != nil {
				return err
			}

			if entry == nil {
				return fmt.Errorf("missing entry %q", entryPath)
			}

			data := credentialStorageEntry{}
			err = entry.DecodeJSON(&data)
			if err != nil {
				return fmt.Errorf("invalid entry %q: %w", entryPath, err)
			}

			credentials = append(credentials, protocol.CredentialDescriptor{
				Type:         protocol.CredentialType(data.CredentialType),
				CredentialID: data.CredentialId,
			})
		}

		return nil
	})

	return credentials, err
}

func (b *backend) pathInternalLoginChallengeRead(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
	otp, err := roottoken.GenerateOTP(0)
	if err != nil {
		return nil, logical.CodedError(http.StatusInternalServerError, err.Error())
	}

	publicKeyRequest := protocol.PublicKeyCredentialRequestOptions{
		Challenge:      wrapChallenge(b.loginChallengeSalt, otp),
		RelyingPartyID: b.config.Load().RelyingPartyId,
		// To make it two-factor, UserVerification is required
		UserVerification: protocol.VerificationRequired,
	}

	return &logical.Response{
		Data: map[string]any{
			"publicKey": publicKeyRequest,
		},
	}, nil
}

func (b *backend) pathInternalLoginFinish(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
	response, ok := data.GetOk("assertion")
	if !ok {
		return nil, logical.CodedError(http.StatusBadRequest, "assertion is required")
	}

	resp, err := protocol.ParseCredentialRequestResponseBytes([]byte(response.(string)))
	if err != nil {
		return nil, err
	}

	_, challenge, err := unwrapChallenge(b.loginChallengeSalt, resp.Response.CollectedClientData.Challenge)
	if err != nil {
		return nil, err
	}

	aliasName := string(resp.Response.UserHandle)
	entry, err := req.Storage.Get(ctx, escapedPathJoin("v1", "credentials", aliasName, resp.ID))
	if err != nil {
		return nil, logical.CodedError(http.StatusInternalServerError, err.Error())
	}

	if entry == nil {
		return nil, logical.CodedError(http.StatusUnauthorized, "key not allowed")
	}

	decodedEntry := credentialStorageEntry{}
	err = entry.DecodeJSON(&decodedEntry)
	if err != nil {
		return nil, logical.CodedError(http.StatusInternalServerError, "invalid entry %q: %v", resp.ID, err)
	}

	config := b.config.Load()
	err = resp.Verify(
		resp.Response.CollectedClientData.Challenge,
		config.RelyingPartyId,
		"",
		config.RelyingPartyOrigins,
		nil,
		nil,
		protocol.TopOriginAutoVerificationMode,
		false,
		true,
		true,
		decodedEntry.CredentialBytes,
		protocol.SignaturePolicy{},
	)
	if err != nil {
		return nil, logical.CodedError(http.StatusUnauthorized, err.Error())
	}

	err = consumeChallenge(ctx, req.Storage, challenge)
	if err != nil {
		if _, ok := errors.AsType[logical.HTTPCodedError](err); ok {
			return nil, err
		}
		return nil, logical.CodedError(http.StatusInternalServerError, err.Error())
	}

	return &logical.Response{Auth: &logical.Auth{
		Alias: &logical.Alias{
			Name: aliasName,
		},
	}}, nil
}
