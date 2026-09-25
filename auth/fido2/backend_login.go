// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package fido2

import (
	"context"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"net/http"
	"strings"

	"github.com/openbao/openbao/sdk/v2/framework"
	"github.com/openbao/openbao/sdk/v2/helper/roottoken"
	"github.com/openbao/openbao/sdk/v2/logical"

	"github.com/go-webauthn/webauthn/protocol"
)

func (b *backend) pathInternalLoginChallengeRead(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
	entityID, ok := data.GetOk("entity_id")
	if !ok {
		return nil, logical.CodedError(http.StatusBadRequest, "entity_id is required")
	}

	entity, err := b.System().EntityInfo(entityID.(string))
	if err != nil {
		return nil, err
	}
	if entity == nil {
		return nil, logical.CodedError(http.StatusNotFound, "entity not found")
	}

	allowedCredentials := []protocol.CredentialDescriptor{}
	var alias *logical.Alias
	for _, alias = range entity.Aliases {
		if alias.MountAccessor == req.MountAccessor {
			continue
		}
		id, ok := alias.Metadata[metadataKeyCredentialID]
		if !ok {
			continue
		}
		credType, ok := alias.Metadata[metadataKeyCredentialType]
		if !ok {
			continue
		}

		rawId, err := base64.RawURLEncoding.DecodeString(id)
		if err != nil {
			return nil, err
		}

		allowedCredentials = append(allowedCredentials, protocol.CredentialDescriptor{
			CredentialID: protocol.URLEncodedBase64(rawId),
			Type:         protocol.CredentialType(credType),
		})
	}

	if alias == nil {
		return nil, fmt.Errorf("no alias available for self-service, please ask your authentication admin to create a fresh alias for mount-accessor %q and entity %q", req.MountAccessor, req.EntityID)
	}

	otp, err := roottoken.GenerateOTP(0)
	if err != nil {
		return nil, err
	}

	hmac := b.loginChallengeSalt.GetHMAC(otp)

	challenge := fmt.Sprintf("%s\x00%s", otp, hmac) // TODO: bind the challenge to a user

	publicKeyRequest := protocol.PublicKeyCredentialRequestOptions{
		Challenge:          protocol.URLEncodedBase64(challenge),
		AllowedCredentials: allowedCredentials,
		RelyingPartyID:     "localhost",
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

	otp, actualHmac, ok := strings.CutLast(resp.Response.CollectedClientData.Challenge, "\x00")
	if ok && subtle.ConstantTimeCompare([]byte(b.loginChallengeSalt.GetHMAC(otp)), []byte(actualHmac)) == 0 {
		return nil, logical.CodedError(http.StatusBadRequest, "invalid challenge")
	}

	entityID, ok := data.GetOk("entity_id")
	if !ok {
		return nil, logical.CodedError(http.StatusBadRequest, "entity_id is required")
	}

	entity, err := b.System().EntityInfo(entityID.(string))
	if err != nil {
		return nil, err
	}
	if entity == nil {
		return nil, logical.CodedError(http.StatusNotFound, "entity not found")
	}

	var alias *logical.Alias
	for _, alias = range entity.Aliases {
		if alias.MountAccessor == req.MountAccessor {
			continue
		}
		id, ok := alias.Metadata[metadataKeyCredentialID]
		if !ok {
			continue
		}
		if id == resp.ID {
			break
		}
	}

	if alias == nil {
		return nil, logical.CodedError(http.StatusUnauthorized, "key not allowed")
	}

	credentialBytes, ok := alias.Metadata[metadataKeyCredentialBytes]
	if !ok {
		return nil, logical.CodedError(http.StatusInternalServerError, "invalid alias metadata")
	}

	credentialBytesRaw, err := base64.RawURLEncoding.DecodeString(credentialBytes)
	if err != nil {
		return nil, logical.CodedError(http.StatusInternalServerError, "invalid alias metadata: %v", err)
	}

	err = resp.Verify(resp.Response.CollectedClientData.Challenge, "localhost", "", []string{"http://localhost:8200"}, nil, nil, protocol.TopOriginAutoVerificationMode, false, false, false, credentialBytesRaw, protocol.SignaturePolicy{})
	if err != nil {
		return nil, logical.CodedError(http.StatusUnauthorized, err.Error())
	}

	return &logical.Response{Auth: &logical.Auth{
		Alias: alias,
	}}, nil
}
