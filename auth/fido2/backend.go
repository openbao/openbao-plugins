// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package fido2

import (
	"context"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"path"
	"slices"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
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

	configLock sync.Mutex // protects against concurrent write/patch operations, reads are protected using atomics
	config     atomic.Pointer[config]

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
			Pattern:      "config",
			HelpSynopsis: "configuration of the auth method (e.g relying party ID)",
			Fields: map[string]*framework.FieldSchema{
				"relying_party_id": {
					Type:     framework.TypeString,
					Required: true,
				},
				"relying_party_origins": {
					Type:     framework.TypeCommaStringSlice,
					Required: true,
				},
			},
			Operations: map[logical.Operation]framework.OperationHandler{
				logical.ReadOperation: &framework.PathOperation{
					Callback: b.pathConfigRead,
				},
				logical.UpdateOperation: &framework.PathOperation{
					Callback: b.pathConfigWrite,
				},
			},
		}, {
			Pattern:      "internal/login/challenge",
			HelpSynopsis: "Warning: this API is not stable and only meant for consumption via the bundled UI",
			Fields:       map[string]*framework.FieldSchema{},
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
			Pattern:      "self-service-enroll",
			HelpSynopsis: "Get an enrollment token for the current identity",
			Operations: map[logical.Operation]framework.OperationHandler{
				logical.UpdateOperation: &framework.PathOperation{
					Callback: b.pathSelfServiceEnroll,
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

	err = b.loadConfig(ctx, req.Storage)
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

type credentialStorageEntry struct {
	CredentialId    protocol.URLEncodedBase64
	CredentialType  string
	CredentialBytes protocol.URLEncodedBase64
	Raw             string
}

func (b *backend) pathInternalEnrollResponseWrite(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
	// https://fidoalliance.org/specs/fidoserver/fido-server-v2.3-rd-20260226.html

	response, ok := data.GetOk("assertion")
	if !ok {
		return nil, logical.CodedError(http.StatusBadRequest, "assertion is required")
	}

	assertion, err := protocol.ParseCredentialCreationResponseBytes([]byte(response.(string)))
	if err != nil {
		return nil, logical.CodedError(http.StatusBadRequest, "invalid assertion: %v", err)
	}

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

	config := b.config.Load()
	_, err = assertion.Verify(
		assertion.Response.CollectedClientData.Challenge, // Bypass the challenge validation: we already validated it above
		config.RelyingPartyId,
		config.RelyingPartyOrigins,
		nil,
		nil,
		protocol.TopOriginAutoVerificationMode,
		false,
		false,
		false,
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

	otp, aliasName, err := splitToken(token)
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
		otpPath := path.Join("v1", "tokens", aliasName, otp)
		val, err := s.Get(ctx, otpPath)
		if err != nil {
			return err
		}
		if val != nil {
			return logical.CodedError(http.StatusForbidden, "token reuse detected")
		}

		err = s.Put(ctx, &logical.StorageEntry{
			Key:   otpPath,
			Value: []byte("{}"),
		})
		if err != nil {
			return err
		}

		return s.Put(ctx, &logical.StorageEntry{
			Key:   path.Join("v1", "credentials", aliasName, assertion.ID),
			Value: storageEntryValue,
		})
	})
	if err != nil {
		if _, ok := errors.AsType[logical.HTTPCodedError](err); ok {
			return nil, err
		}
		return nil, logical.CodedError(http.StatusInternalServerError, err.Error())
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
	token, ok := data.GetOk("token")
	if !ok {
		return nil, logical.CodedError(http.StatusBadRequest, "token is required")
	}

	challenge := fmt.Sprintf("%d\x00%s", time.Now().Unix(), token.(string))

	hmac := b.enrollChallengeSalt.GetHMAC(challenge)

	challenge = fmt.Sprintf("%s\x00%s", challenge, hmac)

	_, aliasName, err := splitToken(token.(string))
	if err != nil {
		return nil, logical.CodedError(http.StatusBadRequest, err.Error())
	}

	existingCredentials, err := findCredentialsForAlias(ctx, req.Storage, aliasName)
	if err != nil {
		return nil, logical.CodedError(http.StatusInternalServerError, err.Error())
	}

	config := b.config.Load()
	publicKeyRequest := protocol.PublicKeyCredentialCreationOptions{
		Challenge: protocol.URLEncodedBase64(challenge),
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
		CredentialExcludeList: existingCredentials, // TODO: this is not needed for Passkeys, we might want to support non-discoverable credentials in the future, so lets keep it for now
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
		},
	}

	return &logical.Response{
		Data: map[string]any{
			"publicKey": publicKeyRequest,
		},
	}, nil
}

func (b *backend) pathSelfServiceEnroll(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
	entity, err := b.System().EntityInfo(req.EntityID)
	if err != nil {
		return nil, err
	}
	if entity == nil {
		return nil, logical.CodedError(http.StatusNotFound, "entity not found")
	}

	aliasIdx := slices.IndexFunc(entity.Aliases, func(alias *logical.Alias) bool {
		return alias.MountAccessor == req.MountAccessor
	})

	if aliasIdx == -1 {
		return nil, fmt.Errorf("no alias available for self-service, please ask your authentication admin to create an alias for mount-accessor %q and entity %q", req.MountAccessor, req.EntityID)
	}

	otp, err := roottoken.GenerateOTP(16)
	if err != nil {
		return nil, err
	}

	token := fmt.Sprintf("v1.%s.%s", otp, entity.Aliases[aliasIdx].Name)

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
