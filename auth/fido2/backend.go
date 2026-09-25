// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package fido2

import (
	"context"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/openbao/openbao-plugins/auth/fido2/ui"
	"github.com/openbao/openbao/sdk/v2/framework"
	"github.com/openbao/openbao/sdk/v2/helper/roottoken"
	"github.com/openbao/openbao/sdk/v2/helper/salt"
	"github.com/openbao/openbao/sdk/v2/logical"

	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/protocol/webauthncose"
)

var pluginVersion string

type backend struct {
	*framework.Backend

	enrollChallengeSalt *salt.Salt
	loginChallengeSalt  *salt.Salt
}

// Factory returns a new backend as logical.Backend.
func Factory(ctx context.Context, conf *logical.BackendConfig) (logical.Backend, error) {
	b := Backend()
	if err := b.Setup(ctx, conf); err != nil {
		return nil, err
	}
	return b, nil
}

func Backend() *backend {
	b := &backend{}

	b.Backend = &framework.Backend{
		//TODO: AuthRenew:   b.pathLoginRenew,
		BackendType: logical.TypeCredential,
		Help:        backendHelp,
		PathsSpecial: &logical.Paths{
			Unauthenticated: []string{
				"login",
				"internal/login/challenge",
				"internal/login/finish",
				"internal/enroll/challenge",
				"internal/enroll/assertion",
				"internal/ui/*",
			},
			SealWrapStorage: []string{
				"config",
			},
		},
		Paths: []*framework.Path{{
			Pattern:      "internal/login/challenge",
			HelpSynopsis: "Warning: this API is not stable and only meant for consumption via the bundled UI",
			Fields: map[string]*framework.FieldSchema{
				"entity_id": {
					Type:     framework.TypeString,
					Required: true,
					Query:    true,
				},
			},
			Operations: map[logical.Operation]framework.OperationHandler{
				logical.ReadOperation: &framework.PathOperation{
					Unpublished: true,
					Callback:    b.pathInternalLoginChallengeRead,
				},
			},
		}, {
			Pattern:      "internal/login/finish",
			HelpSynopsis: "Warning: this API is not stable and only meant for consumption via the bundled UI",
			Fields: map[string]*framework.FieldSchema{
				"assertion": {
					Type:     framework.TypeString,
					Required: true,
				},
				"entity_id": {
					Type:     framework.TypeString,
					Required: true,
				},
			},
			Operations: map[logical.Operation]framework.OperationHandler{
				logical.UpdateOperation: &framework.PathOperation{
					Unpublished: true,
					Callback:    b.pathInternalLoginFinish,
				},
			},
		}, {
			Pattern:      "internal/enroll/challenge",
			HelpSynopsis: "Warning: this API is not stable and only meant for consumption via the bundled UI",
			Fields: map[string]*framework.FieldSchema{
				"token": {
					Type:     framework.TypeString,
					Required: true,
					Query:    true,
				},
			},
			Operations: map[logical.Operation]framework.OperationHandler{
				logical.ReadOperation: &framework.PathOperation{
					Unpublished: true,
					Callback:    b.pathInternalEnrollChallengeRead,
				},
			},
		}, {
			Pattern:      "internal/enroll/assertion",
			HelpSynopsis: "Warning: this API is not stable and only meant for consumption via the bundled UI",
			Fields: map[string]*framework.FieldSchema{
				"assertion": {
					Type:     framework.TypeString,
					Required: true,
				},
			},
			Operations: map[logical.Operation]framework.OperationHandler{
				logical.UpdateOperation: &framework.PathOperation{
					Unpublished: true,
					Callback:    b.pathInternalEnrollResponseWrite,
				},
			},
		}, {
			Pattern:      "enroll/self-serivce/enroll",
			HelpSynopsis: "Get an enrollment token for the current identity",
			Operations: map[logical.Operation]framework.OperationHandler{
				logical.UpdateOperation: &framework.PathOperation{
					Callback: b.pathSelfService,
				},
			},
		}, {
			Pattern:      "internal/ui/" + framework.MatchAllRegex("path"),
			HelpSynopsis: "Enroll the current identity",
			Fields: map[string]*framework.FieldSchema{
				"path": {
					Type:     framework.TypeString,
					Required: true,
				},
			},
			Operations: map[logical.Operation]framework.OperationHandler{
				logical.ReadOperation: &framework.PathOperation{
					Callback: func(ctx context.Context, r *logical.Request, fd *framework.FieldData) (*logical.Response, error) {
						return ui.PathGet(ctx, r, fd, b.Logger().Named("ui"))
					},
				},
			},
		}},
		InitializeFunc: b.initialize,
		Invalidate:     b.invalidate,
		RunningVersion: pluginVersion,
	}

	return b
}

func (b *backend) initialize(ctx context.Context, req *logical.InitializationRequest) error {
	var err error
	b.loginChallengeSalt, err = salt.NewSalt(ctx, req.Storage, &salt.Config{
		Location: "config/login-challenge-salt",
	})
	if err != nil {
		return err
	}

	b.enrollChallengeSalt, err = salt.NewSalt(ctx, req.Storage, &salt.Config{
		Location: "config/enroll-challenge-salt",
	})
	if err != nil {
		return err
	}

	return nil
}

// ClearCaches deletes all cached clients and credentials.
func (b *backend) ClearCaches() {
	// TODO
}

// invalidate resets the plugin. This is called when a key is updated via
// replication.
func (b *backend) invalidate(_ context.Context, key string) {
	switch key {
	case "config":
		b.ClearCaches()
	}
}

func splitToken(x string) (otp string, aliasName string, err error) {
	x, ok := strings.CutPrefix(x, "v1.")
	if !ok {
		err = errors.New("invalid token: missing or unsupported version")
		return
	}

	otp, aliasName, ok = strings.Cut(x, ".")
	if !ok {
		err = errors.New("invalid token: format error")
		return
	}

	return
}

func splitChallenge(x string) (timestamp int64, token string, hmac string, err error) {
	parts := strings.Split(x, "\x00")
	if len(parts) != 3 {
		err = fmt.Errorf("invalid challenge: expected 3 parts, got %d", len(parts))
		return
	}

	timestamp, err = strconv.ParseInt(parts[0], 10, 64)
	token = parts[1]
	hmac = parts[2]

	return
}

func (b *backend) pathInternalEnrollResponseWrite(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
	// https://fidoalliance.org/specs/fidoserver/fido-server-v2.3-rd-20260226.html

	response, ok := data.GetOk("assertion")
	if !ok {
		return nil, logical.CodedError(http.StatusBadRequest, "assertion is required")
	}

	assertion, err := protocol.ParseCredentialCreationResponseBytes([]byte(response.(string)))
	//assertion, err := webauthn.ParseAssertion(strings.NewReader(response.(string)))
	if err != nil {
		return nil, fmt.Errorf("invalid assertion: %w", err)
	}
	//webauthn.VerifyAttestation(assertion)

	challenge, err := base64.RawURLEncoding.DecodeString(assertion.Response.CollectedClientData.Challenge)
	if err != nil {
		if cie, ok := errors.AsType[base64.CorruptInputError](err); ok {
			err := assertion.Response.CollectedClientData.Challenge + "\n" + strings.Repeat(" ", int(cie-1)) + "^"
			return nil, logical.CodedError(http.StatusBadRequest, err)

		}
		return nil, logical.CodedError(http.StatusBadRequest, "invalid challenge: %v / %q", err, assertion.Response.CollectedClientData.Challenge)
	}

	timestamp, token, hmac, err := splitChallenge(string(challenge))
	if err != nil {
		return nil, logical.CodedError(http.StatusBadRequest, err.Error())
	}

	if time.Now().Sub(time.Unix(timestamp, 0)) > 10*time.Minute {
		return nil, logical.CodedError(http.StatusBadRequest, "challenge expired")
	}

	expectedHmac := b.enrollChallengeSalt.GetHMAC(fmt.Sprintf("%d\x00%s", timestamp, token))
	if subtle.ConstantTimeCompare([]byte(expectedHmac), []byte(hmac)) == 0 {
		return nil, logical.CodedError(http.StatusBadRequest, "invalid challenge")
	}

	_, err = assertion.Verify(assertion.Response.CollectedClientData.Challenge, // Bypass the challenge validation: we already validated it above
		"localhost", []string{"http://localhost:8200"}, nil, nil, protocol.TopOriginAutoVerificationMode, false, false, false, nil, []protocol.CredentialParameter{{Type: protocol.PublicKeyCredentialType, Algorithm: webauthncose.AlgES256}}, protocol.AttestationPolicy{}, protocol.SignaturePolicy{})
	if err != nil {
		return nil, logical.CodedError(http.StatusBadRequest, "invalid assertion: %w", err)
	}

	_, aliasName, err := splitToken(token)
	if err != nil {
		return nil, logical.CodedError(http.StatusBadRequest, "invalid token: %w", err)
	}

	return &logical.Response{Auth: &logical.Auth{
		Alias: &logical.Alias{
			Name: aliasName,
			Metadata: map[string]string{
				metadataKeyCredentialID:    assertion.ID,
				metadataKeyCredentialType:  assertion.Type,
				metadataKeyCredentialBytes: protocol.URLEncodedBase64(assertion.Response.AttestationObject.AuthData.AttData.CredentialPublicKey).String(),
			},
		},
	}}, nil
}

func (b *backend) pathInternalEnrollChallengeRead(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
	token, ok := data.GetOk("token")
	if !ok {
		return nil, logical.CodedError(http.StatusBadRequest, "token is required")
	}

	challenge := fmt.Sprintf("%d\x00%s", time.Now().Unix(), token.(string))

	hmac := b.enrollChallengeSalt.GetHMAC(challenge)

	challenge = fmt.Sprintf("%s\x00%s", challenge, hmac)

	publicKeyRequest := protocol.PublicKeyCredentialCreationOptions{
		Challenge: protocol.URLEncodedBase64(challenge),
		RelyingParty: protocol.RelyingPartyEntity{
			ID: "localhost",
			CredentialEntity: protocol.CredentialEntity{
				Name: "OpenBao",
			},
		},
		User: protocol.UserEntity{
			DisplayName: "Jamie Doe",
			CredentialEntity: protocol.CredentialEntity{
				Name: "jamiedoe",
			},
			ID: protocol.URLEncodedBase64("asdf"), // TODO: entity id
		},
		// Attestation: protocol.PreferDirectAttestation, TODO: support this
		Parameters: []protocol.CredentialParameter{{
			Type:      protocol.PublicKeyCredentialType,
			Algorithm: webauthncose.AlgES256,
		}},
	}

	return &logical.Response{
		Data: map[string]any{
			"publicKey": publicKeyRequest,
		},
	}, nil
}

const (
	metadataKeyCredentialID    = "fido2_credential_id"
	metadataKeyCredentialType  = "fido2_credential_type"
	metadataKeyCredentialBytes = "fido2_credential_bytes"
)

func (b *backend) pathSelfService(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
	entity, err := b.System().EntityInfo(req.EntityID)
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
		_, ok := alias.Metadata[metadataKeyCredentialID]
		if ok {
			continue // alias has already been enrolled
		}
	}

	if alias == nil {
		return nil, fmt.Errorf("no alias available for self-service, please ask your authentication admin to create a fresh alias for mount-accessor %q and entity %q", req.MountAccessor, req.EntityID)
	}

	otp, err := roottoken.GenerateOTP(0)
	if err != nil {
		return nil, err
	}

	token := fmt.Sprintf("v1.%s.%s", otp, alias.Name)

	// TODO: persist

	// TODO: we might want to generate a lease here: gives us revoke handling (manual or time based) for free

	return &logical.Response{
		Data: map[string]any{
			"token": token,
		},
	}, nil
}

const backendHelp = `
TODO`
