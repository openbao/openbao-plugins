// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package fido2

import (
	"context"
	"crypto/subtle"
	"fmt"
	"net/http"
	"path"
	"strings"

	"github.com/openbao/openbao/sdk/v2/framework"
	"github.com/openbao/openbao/sdk/v2/helper/roottoken"
	"github.com/openbao/openbao/sdk/v2/logical"

	"github.com/go-webauthn/webauthn/protocol"
)

func findCredentialsForAlias(ctx context.Context, s logical.Storage, aliasName string) ([]protocol.CredentialDescriptor, error) {
	var credentials []protocol.CredentialDescriptor
	err := logical.WithTransaction(ctx, s, func(s logical.Storage) error {
		basePath := path.Join("v1", "credentials", aliasName) + "/"
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
			entry.DecodeJSON(&data)
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
	aliasName, ok := data.GetOk("alias")
	if !ok {
		return nil, logical.CodedError(http.StatusBadRequest, "alias is required")
	}

	allowedCredentials, err := findCredentialsForAlias(ctx, req.Storage, aliasName.(string))
	if err != nil {
		return nil, logical.CodedError(http.StatusInternalServerError, err.Error())
	}

	if len(allowedCredentials) == 0 {
		return nil, fmt.Errorf("no credential enrolled for this alias")
	}

	otp, err := roottoken.GenerateOTP(0)
	if err != nil {
		return nil, logical.CodedError(http.StatusInternalServerError, err.Error())
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

	aliasName, ok := data.GetOk("alias")
	if !ok {
		return nil, logical.CodedError(http.StatusBadRequest, "alias is required")
	}

	entry, err := req.Storage.Get(ctx, path.Join("v1", "credentials", aliasName.(string), resp.ID))
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

	err = resp.Verify(resp.Response.CollectedClientData.Challenge, "localhost", "", []string{"http://localhost:8200"}, nil, nil, protocol.TopOriginAutoVerificationMode, false, false, false, decodedEntry.CredentialBytes, protocol.SignaturePolicy{})
	if err != nil {
		return nil, logical.CodedError(http.StatusUnauthorized, err.Error())
	}

	return &logical.Response{Auth: &logical.Auth{
		Alias: &logical.Alias{
			Name: aliasName.(string),
		},
	}}, nil
}
