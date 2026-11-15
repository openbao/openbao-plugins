// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package fido2

import (
	"context"
	"errors"
	"path"
	"strings"

	"github.com/openbao/openbao-plugins/auth/fido2/helper/invalidate"
	"github.com/openbao/openbao-plugins/auth/fido2/ui"
	"github.com/openbao/openbao/sdk/v2/framework"
	"github.com/openbao/openbao/sdk/v2/helper/salt"
	"github.com/openbao/openbao/sdk/v2/logical"

	"github.com/go-webauthn/webauthn/protocol"
)

var pluginVersion string

type backend struct {
	*framework.Backend

	config invalidate.Invalidateable[config]

	enrollChallengeSalt invalidate.Invalidateable[salt.Salt]
	loginChallengeSalt  invalidate.Invalidateable[salt.Salt]
}

// Factory returns a new backend as logical.Backend.
func Factory(ctx context.Context, conf *logical.BackendConfig) (logical.Backend, error) {
	b := Backend()
	if err := b.Setup(ctx, conf); err != nil {
		return nil, err
	}
	return b, nil
}

const (
	SecretTypeEnrollmentToken = "enrollment_token"
)

func Backend() *backend {
	b := &backend{}

	b.Backend = &framework.Backend{
		//TODO: AuthRenew:   b.pathLoginRenew,
		BackendType: logical.TypeCredential,
		Help:        backendHelp,
		PathsSpecial: &logical.Paths{
			Unauthenticated: []string{
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
				"recommend_hardware_token": {
					Type:        framework.TypeBool,
					Description: "Whether or not the users should be warned when trying to enroll a software token. Notice: this will not enforce the use of hardware tokens.",
					Default:     false,
					Required:    false,
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
			Pattern:      "self-service/enroll",
			HelpSynopsis: "Get an enrollment token for the current identity",
			Operations: map[logical.Operation]framework.OperationHandler{
				logical.UpdateOperation: &framework.PathOperation{
					Callback: b.pathSelfServiceEnroll,
				},
			},
		}, {
			Pattern:      "self-service/keys/",
			HelpSynopsis: "Get enrolled keys for current user",
			Operations: map[logical.Operation]framework.OperationHandler{
				logical.ListOperation: &framework.PathOperation{
					Callback: b.pathSelfServiceList,
				},
			},
		}, {
			Pattern: "self-service/keys/" + framework.GenericNameRegex("id"),
			Fields: map[string]*framework.FieldSchema{
				"id": &framework.FieldSchema{
					Type:     framework.TypeString,
					Required: true,
				},
				"name": &framework.FieldSchema{
					Type:     framework.TypeString,
					Required: false,
				},
			},
			HelpSynopsis: "Mange enrolled keys for current user",
			Operations: map[logical.Operation]framework.OperationHandler{
				logical.ReadOperation: &framework.PathOperation{
					Callback: b.pathSelfServiceRead,
				},
				logical.DeleteOperation: &framework.PathOperation{
					Callback: b.pathSelfServiceDelete,
				},
				logical.PatchOperation: &framework.PathOperation{
					Callback: b.pathSelfServicePatch,
				},
			},
		}, {
			Pattern: "manage/" + framework.GenericNameRegex("alias") + "/keys/",
			Fields: map[string]*framework.FieldSchema{
				"alias": &framework.FieldSchema{
					Type:     framework.TypeString,
					Required: true,
				},
			},
			HelpSynopsis: "Get enrolled keys for given alias",
			Operations: map[logical.Operation]framework.OperationHandler{
				logical.ListOperation: &framework.PathOperation{
					Callback: b.pathAdminList,
				},
			},
		}, {
			Pattern: "manage/" + framework.GenericNameRegex("alias") + "/keys/" + framework.GenericNameRegex("id"),
			Fields: map[string]*framework.FieldSchema{
				"alias": &framework.FieldSchema{
					Type:     framework.TypeString,
					Required: true,
				},
				"id": &framework.FieldSchema{
					Type:     framework.TypeString,
					Required: true,
				},
				"name": &framework.FieldSchema{
					Type:     framework.TypeString,
					Required: false,
				},
			},
			HelpSynopsis: "Mange enrolled keys given alias",
			Operations: map[logical.Operation]framework.OperationHandler{
				logical.ReadOperation: &framework.PathOperation{
					Callback: b.pathAdminRead,
				},
				logical.DeleteOperation: &framework.PathOperation{
					Callback: b.pathAdminDelete,
				},
				logical.PatchOperation: &framework.PathOperation{
					Callback: b.pathAdminPatch,
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
		Secrets: []*framework.Secret{{
			Type: SecretTypeEnrollmentToken,
			Fields: map[string]*framework.FieldSchema{
				"token": {
					Type:        framework.TypeString,
					Description: "enrollment token",
				},
			},
			Revoke: b.revokeEnrollmentToken,
		}},
		InitializeFunc: b.initialize,
		Invalidate:     b.invalidate,
		PeriodicFunc:   b.periodic,

		RunningVersion: pluginVersion,
	}

	return b
}

const (
	saltPrefix              = "salts/v1/"
	loginChallengeSaltPath  = saltPrefix + "login-challenge"
	enrollChallengeSaltPath = saltPrefix + "enroll-challenge"
	configStoragePath       = "config/v1"
)

func (b *backend) initialize(ctx context.Context, req *logical.InitializationRequest) error {
	b.loginChallengeSalt = invalidate.NewInvalidateableSalt(loginChallengeSaltPath)
	_, err := b.loginChallengeSalt.Get(ctx, req.Storage)
	if err != nil {
		return err
	}

	b.enrollChallengeSalt = invalidate.NewInvalidateableSalt(enrollChallengeSaltPath)
	_, err = b.enrollChallengeSalt.Get(ctx, req.Storage)
	if err != nil {
		return err
	}

	b.config = invalidate.NewInvalidateable(b.loadConfig)
	_, err = b.config.Get(ctx, req.Storage)
	if err != nil {
		return err
	}

	return nil
}

func (b *backend) periodic(ctx context.Context, req *logical.Request) error {
	return cleanupExpiredConsumedChallenges(ctx, b.Logger().Named("cleanupExpiredConsumedChallenges"), req.Storage)
}

// invalidate is called on standbys when the primary updated the storage
func (b *backend) invalidate(_ context.Context, key string) {
	switch key {
	case configStoragePath:
		b.config.Invalidate()
	case enrollChallengeSaltPath:
		b.enrollChallengeSalt.Invalidate()
	case loginChallengeSaltPath:
		b.loginChallengeSalt.Invalidate()
	default:
		b.Logger().Warn("unknown key during invalidation", "key", key)
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

type credentialStorageEntry struct {
	CredentialId    protocol.URLEncodedBase64
	CredentialType  string
	CredentialBytes protocol.URLEncodedBase64
	Raw             string
	DisplayName     string
}

var pathReplacer = strings.NewReplacer(
	"%", "%25",
	".", "%2E",
	"/", "%2F",
)

func escapedPathJoin(parts ...string) string {
	encodedParts := make([]string, len(parts))
	for i := range parts {
		encodedParts[i] = pathReplacer.Replace(parts[i])
	}
	return path.Join(encodedParts...)
}

func (b *backend) revokeEnrollmentToken(ctx context.Context, req *logical.Request, d *framework.FieldData) (*logical.Response, error) {
	path := req.Secret.InternalData["storage_path"].(string)
	return nil, req.Storage.Delete(ctx, path)
}

const backendHelp = `
TODO`
