// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package fido2

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/openbao/openbao/sdk/v2/framework"
	"github.com/openbao/openbao/sdk/v2/logical"

	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/protocol/webauthncose"
)

func (b *backend) pathInternalEnrollResponseWrite(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
	// https://fidoalliance.org/specs/fidoserver/fido-server-v2.3-rd-20260226.html

	config, err := b.config.Get(ctx, req.Storage)
	if err != nil {
		return nil, logical.CodedError(http.StatusInternalServerError, err.Error())
	}

	salt, err := b.enrollChallengeSalt.Get(ctx, req.Storage)
	if err != nil {
		return nil, logical.CodedError(http.StatusInternalServerError, err.Error())
	}

	response, ok := data.GetOk("assertion")
	if !ok {
		return nil, logical.CodedError(http.StatusBadRequest, "assertion is required")
	}

	assertion, err := protocol.ParseCredentialCreationResponseBytes([]byte(response.(string)))
	if err != nil {
		return nil, logical.CodedError(http.StatusBadRequest, "invalid assertion: %v", err)
	}

	token, challenge, err := unwrapChallenge(salt, assertion.Response.CollectedClientData.Challenge)
	if err != nil {
		return nil, err
	}

	_, err = assertion.Verify(
		assertion.Response.CollectedClientData.Challenge, // Bypass the challenge validation: we already validated it above
		config.RelyingPartyId,
		config.RelyingPartyOrigins,
		nil,
		nil,
		protocol.TopOriginAutoVerificationMode,
		false,
		true,
		true,
		nil,
		[]protocol.CredentialParameter{{Type: protocol.PublicKeyCredentialType, Algorithm: webauthncose.AlgES256}},
		protocol.AttestationPolicy{},
		protocol.SignaturePolicy{},
	)
	if err != nil {
		if err, ok := errors.AsType[*protocol.Error](err); ok {
			b.Logger().Debug("protocol error", "type", err.Type, "details", err.Details, "dev_info", err.DevInfo)
		}
		return nil, logical.CodedError(http.StatusBadRequest, "invalid assertion: %v", err)
	}

	_, aliasName, err := splitToken(token)
	if err != nil {
		return nil, logical.CodedError(http.StatusBadRequest, err.Error())
	}

	storageEntryValue, err := json.Marshal(credentialStorageEntry{
		CredentialId:    assertion.RawID,
		CredentialType:  assertion.Type,
		CredentialBytes: assertion.Response.AttestationObject.AuthData.AttData.CredentialPublicKey,
		Raw:             response.(string),
	})
	if err != nil {
		return nil, logical.CodedError(http.StatusInternalServerError, err.Error())
	}

	err = logical.WithTransaction(ctx, req.Storage, func(s logical.Storage) error {
		err := consumeChallenge(ctx, s, challenge)
		if err != nil {
			return err
		}

		return s.Put(ctx, &logical.StorageEntry{
			Key:   escapedPathJoin("v1", "credentials", aliasName, assertion.ID), // the ID might be 1023 bytes long + base64 overhead https://www.w3.org/TR/webauthn-3/#credential-id TODO: is this a problem
			Value: storageEntryValue,
		})
	})
	if err != nil {
		return nil, asCodedError(err)
	}

	return &logical.Response{
		Auth: &logical.Auth{
			Alias: &logical.Alias{
				Name: aliasName,
			},
		},
	}, nil
}

func (b *backend) pathInternalEnrollChallengeRead(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
	config, err := b.config.Get(ctx, req.Storage)
	if err != nil {
		return nil, logical.CodedError(http.StatusInternalServerError, err.Error())
	}

	salt, err := b.enrollChallengeSalt.Get(ctx, req.Storage)
	if err != nil {
		return nil, logical.CodedError(http.StatusInternalServerError, err.Error())
	}

	token, ok := data.GetOk("token")
	if !ok {
		return nil, logical.CodedError(http.StatusBadRequest, "token is required")
	}

	_, aliasName, err := splitToken(token.(string))
	if err != nil {
		return nil, logical.CodedError(http.StatusBadRequest, err.Error())
	}

	rollback, err := logical.StartTxStorage(ctx, req)
	if err != nil {
		return nil, logical.CodedError(http.StatusInternalServerError, err.Error())
	}
	defer rollback()

	tokenStoragePath := escapedPathJoin("v1", "enrollment-tokens", aliasName, token.(string))
	tokenEntry, err := req.Storage.Get(ctx, tokenStoragePath)
	if err != nil {
		return nil, logical.CodedError(http.StatusInternalServerError, err.Error())
	}
	if tokenEntry == nil {
		return nil, logical.CodedError(http.StatusNotFound, "token not found")
	}
	err = req.Storage.Delete(ctx, tokenStoragePath)
	if err != nil {
		return nil, logical.CodedError(http.StatusInternalServerError, err.Error())
	}

	existingCredentials, err := findCredentialsForAlias(ctx, req.Storage, aliasName)
	if err != nil {
		return nil, logical.CodedError(http.StatusInternalServerError, err.Error())
	}

	publicKeyRequest := protocol.PublicKeyCredentialCreationOptions{
		Challenge: wrapChallenge(salt, token.(string)),
		RelyingParty: protocol.RelyingPartyEntity{
			ID: config.RelyingPartyId,
			CredentialEntity: protocol.CredentialEntity{
				Name: "OpenBao",
			},
		},
		User: protocol.UserEntity{
			DisplayName: aliasName,
			Name:        aliasName, // TODO: alias@clustername?
			ID:          protocol.URLEncodedBase64(aliasName),
		},
		CredentialExcludeList: existingCredentials,
		// Attestation: protocol.PreferDirectAttestation, TODO: support this
		Parameters: []protocol.CredentialParameter{{
			Type:      protocol.PublicKeyCredentialType,
			Algorithm: webauthncose.AlgES256,
		}},
		AuthenticatorSelection: protocol.AuthenticatorSelection{
			// Passkeys are "discoverable" (f.k.a "resident") credentials
			// https://www.w3.org/TR/webauthn-3/#discoverable-credential
			ResidentKey:        "required",
			RequireResidentKey: new(true),
			// To make it two-factor, UserVerification is required
			UserVerification: protocol.VerificationRequired,
		},
	}

	if config.RecommendHardwareToken {
		// as of today, these values seem to be largely ignored by browsers
		// we implement our own checks in JavaScript (no enforcement, just recommendation)
		publicKeyRequest.Hints = append(publicKeyRequest.Hints, protocol.PublicKeyCredentialHintSecurityKey)
		publicKeyRequest.AuthenticatorSelection.AuthenticatorAttachment = protocol.CrossPlatform
	}

	err = logical.EndTxStorage(ctx, req)
	if err != nil {
		return nil, logical.CodedError(http.StatusInternalServerError, err.Error())
	}

	return &logical.Response{
		Data: map[string]any{
			"publicKey":                publicKeyRequest,
			"recommend_hardware_token": config.RecommendHardwareToken,
		},
	}, nil
}
