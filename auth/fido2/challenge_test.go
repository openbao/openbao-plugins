// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package fido2

import (
	"fmt"
	"testing"
	"testing/synctest"
	"time"

	"github.com/hashicorp/go-hclog"
	"github.com/openbao/openbao/sdk/v2/helper/salt"
	"github.com/openbao/openbao/sdk/v2/logical"
	"github.com/stretchr/testify/require"
)

func TestConsumedChallengesStressTest(t *testing.T) {
	t.Parallel()

	const (
		m = 42
		n = 100_000 - m
	)

	synctest.Test(t, func(t *testing.T) {
		storage := &logical.InmemStorage{}

		s := salt.NewNonpersistentSalt()

		for i := range n {
			c := wrapChallenge(s, fmt.Sprintf("%09d", i))
			require.NoError(t, consumeChallenge(t.Context(), storage, string(c)))
			time.Sleep(time.Second)
		}

		time.Sleep(5*time.Minute + time.Hour + 1) // 5 minutes token validity + 1 hour clock skew + 1

		for i := range m {
			c := wrapChallenge(s, fmt.Sprintf("%09d", n+i))
			require.NoError(t, consumeChallenge(t.Context(), storage, string(c)))
		}

		logger := hclog.Default()
		logger.SetLevel(hclog.Trace)
		cleanupExpiredConsumedChallenges(t.Context(), logger, storage)

		count, err := logical.CountKeys(t.Context(), logical.NewStorageView(storage, "v1/consumed-challenges"))
		require.NoError(t, err)
		require.Equal(t, m, count)
	})
}

func TestChallengeEncoding(t *testing.T) {
	s := salt.NewNonpersistentSalt()

	payload := "test" + challengeSeparator + "1234"

	wrapped := wrapChallenge(s, payload)

	unwrapped, decoded, err := unwrapChallenge(s, wrapped.String())

	require.NoError(t, err)
	require.Equal(t, payload, unwrapped)
	require.Equal(t, decoded, string(wrapped))

	_, _, err = unwrapChallenge(salt.NewNonpersistentSalt(), wrapped.String())
	require.Error(t, err)
}
