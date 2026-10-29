// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package fido2

import (
	"encoding/json"
	"net/url"
	"path"
	"strings"
	"testing"
	"testing/synctest"
	"time"

	"github.com/descope/virtualwebauthn"
	"github.com/go-webauthn/webauthn/protocol"
	"github.com/hashicorp/go-hclog"
	"github.com/openbao/openbao-plugins/internal/logicaltest"
	"github.com/openbao/openbao/sdk/v2/helper/logging"
	"github.com/openbao/openbao/sdk/v2/logical"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPathJoin(t *testing.T) {
	parts := []string{"a", "b", "c/d", "x/../y", "%", "5%20d"}
	joined := escapedPathJoin(parts...)

	for i, part := range parts {
		var head string
		head, joined, _ = strings.Cut(joined, "/")
		assert.Equal(t, url.PathEscape(part), head, "part %d", i) // this should not be true in the general case (as e.g. + will not be encoded), see case below
		t.Logf("%d expect %q == %q // remaining %q %v", i, head, part, joined, parts)
	}

	assert.Empty(t, joined)

	nonEncoded := []string{"a", "asdf", "+", "$", "?&@$*&#"}
	assert.Equal(t, path.Join(nonEncoded...), escapedPathJoin(nonEncoded...)) // all chars without special meaning in storage paths => should be equivalent path.Join
}

func createBackend(t *testing.T) (*backend, *logical.InmemStorage) {
	storage := &logical.InmemStorage{}

	config := &logical.BackendConfig{
		Logger: logging.NewVaultLogger(hclog.Trace),
		System: &logical.StaticSystemView{
			EntityVal: &logical.Entity{
				Aliases: []*logical.Alias{{
					MountAccessor: "",
					Name:          "test",
				}},
			},
		},
		StorageView: storage,
	}

	b, err := Factory(t.Context(), config)
	require.NoError(t, err)

	require.NoError(t, b.Initialize(t.Context(), &logical.InitializationRequest{
		Storage: storage,
	}))

	return b.(*backend), storage
}

func TestSelfEnroll(t *testing.T) {
	t.Parallel()

	backend, storage := createBackend(t)

	var enrollmentToken string

	stepGetTokenSelf := logicaltest.TestStep{
		Operation: logical.UpdateOperation,
		Path:      "self-service-enroll",
		Check: func(r *logical.Response) error {
			require.NotPanics(t, func() {
				enrollmentToken = r.Data["token"].(string)
			})
			return nil
		},
	}
	stepGetChallenge := logicaltest.TestStep{
		Operation: logical.ReadOperation,
		Path:      "internal/enroll/challenge",
		Data:      map[string]any{},
		PreFlight: func(r *logical.Request) error {
			r.Data["token"] = enrollmentToken
			return nil
		},
	}

	stepGetChallengeFailure := stepGetChallenge
	stepGetChallengeFailure.ErrorRequired = true
	stepGetChallengeFailure.ErrCheck = func(r *logical.Response, err error) error {
		require.Equal(t, "token not found", err.Error())
		return nil
	}

	logicaltest.Test(t, logicaltest.TestCase{
		Backend: backend,
		Storage: storage,
		Steps: []logicaltest.TestStep{
			// get token, use twice -> 2nd should fail
			stepGetTokenSelf,
			stepGetChallenge,
			stepGetChallengeFailure,
		},
	})

	{
		// The `logicaltest.Test` automatically tracks all leases and revokes them after the test
		// We (ab)use this fact to test our revoke handling

		logicaltest.Test(t, logicaltest.TestCase{
			Backend: backend,
			Storage: storage,
			Steps: []logicaltest.TestStep{
				stepGetTokenSelf,
			},
		})
		logicaltest.Test(t, logicaltest.TestCase{
			Backend: backend,
			Storage: storage,
			Steps: []logicaltest.TestStep{
				stepGetChallengeFailure,
			},
		})
	}
}

func TestLogin(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {

		backend, storage := createBackend(t)
		backend.config.Store(&config{
			RelyingPartyId: "example.com", RelyingPartyOrigins: []string{"https://example.com"},
		})

		var (
			enrollmentToken  string
			pkcCreateOptions protocol.PublicKeyCredentialCreationOptions
			pkcGetOptions    protocol.PublicKeyCredentialRequestOptions
		)
		stepGetTokenSelf := logicaltest.TestStep{
			Operation: logical.UpdateOperation,
			Path:      "self-service-enroll",
			Check: func(r *logical.Response) error {
				require.NotPanics(t, func() {
					enrollmentToken = r.Data["token"].(string)
				})
				return nil
			},
		}
		stepEnrollGetChallenge := logicaltest.TestStep{
			Operation: logical.ReadOperation,
			Path:      "internal/enroll/challenge",
			Data:      map[string]any{},
			PreFlight: func(r *logical.Request) error {
				r.Data["token"] = enrollmentToken
				return nil
			},
			Check: func(r *logical.Response) error {
				require.NotPanics(t, func() {
					pkcCreateOptions = r.Data["publicKey"].(protocol.PublicKeyCredentialCreationOptions)
				})
				return nil
			},
		}

		rp := virtualwebauthn.RelyingParty{Name: "Example Corp", ID: "example.com", Origin: "https://example.com"}
		authenticator := virtualwebauthn.NewAuthenticator()
		credential := virtualwebauthn.NewCredential(virtualwebauthn.KeyTypeEC2)

		stepEnrollFinish := logicaltest.TestStep{
			Operation: logical.UpdateOperation,
			Path:      "internal/enroll/assertion",
			Data:      map[string]any{},
			PreFlight: func(r *logical.Request) error {
				publicKeyRequest, err := json.Marshal(pkcCreateOptions)
				if err != nil {
					return err
				}
				parsedAttestationOptions, err := virtualwebauthn.ParseAttestationOptions(string(publicKeyRequest))
				if err != nil {
					return err
				}
				attestationResponse := virtualwebauthn.CreateAttestationResponse(rp, authenticator, credential, *parsedAttestationOptions)
				authenticator.Options.UserHandle = []byte(parsedAttestationOptions.UserID)

				r.Data["assertion"] = attestationResponse
				return nil
			},
		}

		stepEnrollFinishFailure := stepEnrollFinish
		stepEnrollFinishFailure.ErrorRequired = true
		stepEnrollFinishFailure.ErrCheck = func(r *logical.Response, err error) error {
			require.Equal(t, "challenge reuse detected", err.Error())
			return nil
		}

		stepLoginGetChallenge := logicaltest.TestStep{
			Operation: logical.ReadOperation,
			Path:      "internal/login/challenge",
			Data:      map[string]any{},
			PreFlight: func(r *logical.Request) error {
				r.Data["token"] = enrollmentToken
				return nil
			},
			Check: func(r *logical.Response) error {
				require.NotPanics(t, func() {
					pkcGetOptions = r.Data["publicKey"].(protocol.PublicKeyCredentialRequestOptions)
				})
				return nil
			},
		}

		stepLoginFinish := logicaltest.TestStep{
			Operation: logical.UpdateOperation,
			Path:      "internal/login/finish",
			Data:      map[string]any{},
			PreFlight: func(r *logical.Request) error {
				publicKeyRequest, err := json.Marshal(pkcGetOptions)
				if err != nil {
					return err
				}
				parsedAttestationOptions, err := virtualwebauthn.ParseAssertionOptions(string(publicKeyRequest))
				if err != nil {
					return err
				}
				attestationResponse := virtualwebauthn.CreateAssertionResponse(rp, authenticator, credential, *parsedAttestationOptions)

				r.Data["assertion"] = attestationResponse
				return nil
			},
		}

		stepLoginFinishFailure := stepLoginFinish
		stepLoginFinishFailure.ErrorRequired = true
		stepLoginFinishFailure.ErrCheck = func(r *logical.Response, err error) error {
			require.Equal(t, "challenge reuse detected", err.Error())
			return nil
		}

		logicaltest.Test(t, logicaltest.TestCase{
			Backend: backend,
			Storage: storage,
			Steps: []logicaltest.TestStep{
				stepGetTokenSelf,

				stepEnrollGetChallenge,
				stepEnrollFinish,
				stepEnrollFinishFailure,

				stepLoginGetChallenge,
				stepLoginFinish,
				stepLoginFinishFailure,
			},
		})

		t.Log(time.Now().GoString())
		initialList, err := storage.List(t.Context(), "v1/consumed-challenges/")
		require.NoError(t, err)
		require.NotEmpty(t, initialList)

		require.NoError(t, backend.PeriodicFunc(t.Context(), &logical.Request{
			Storage: storage,
		}))

		t.Log(time.Now().GoString())
		list, err := storage.List(t.Context(), "v1/consumed-challenges/")
		require.NoError(t, err)
		require.Equal(t, initialList, list)

		time.Sleep(5*time.Minute + time.Hour + 1) // 5 minutes token validity + 1 hour clock skew + 1
		t.Log(time.Now().GoString())

		require.NoError(t, backend.PeriodicFunc(t.Context(), &logical.Request{
			Storage: storage,
		}))

		list, err = storage.List(t.Context(), "v1/consumed-challenges/")
		require.NoError(t, err)
		require.Empty(t, list)
	})
}
