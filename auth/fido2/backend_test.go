// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package fido2

import (
	"net/url"
	"path"
	"strings"
	"testing"

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

func createBackend(t *testing.T) (*backend, logical.Storage) {
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

	backend, _ := createBackend(t)

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
	stepGetChallengeFailure.ErrorOk = true
	stepGetChallengeFailure.Check = func(r *logical.Response) error {
		require.Equal(t, "token not found", r.Error())
		return nil
	}

	logicaltest.Test(t, logicaltest.TestCase{
		Backend: backend,
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
			Steps: []logicaltest.TestStep{
				stepGetTokenSelf,
			},
		})
		logicaltest.Test(t, logicaltest.TestCase{
			Backend: backend,
			Steps: []logicaltest.TestStep{
				stepGetChallengeFailure,
			},
		})
	}
}
