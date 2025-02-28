// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package gcpkms

import (
	"context"
	"path"
	"testing"

	"github.com/openbao/openbao/sdk/v2/logical"
)

func TestPathKeysRegister_Write(t *testing.T) {
	t.Run("field_validation", func(t *testing.T) {
		testFieldValidation(t, logical.UpdateOperation, "keys/register/my-key")
	})

	cryptoKey, cleanup := testCreateKMSCryptoKeySymmetric(t)
	defer cleanup()

	cases := []struct {
		name      string
		cryptoKey string
		verify    bool
		err       bool
	}{
		{
			"key_exists_verify",
			cryptoKey,
			true,
			false,
		},
		{
			"key_exists_no_verify",
			cryptoKey,
			false,
			false,
		},
		{
			"key_not_exists_verify",
			"not-a-real-key",
			true,
			true,
		},
		{
			"key_not_exists_no_verify",
			"not-a-real-key",
			false,
			false,
		},
	}

	t.Run("group", func(t *testing.T) {
		for _, tc := range cases {
			tc := tc

			t.Run(tc.name, func(t *testing.T) {
				key := path.Base(tc.cryptoKey)

				b, storage := testBackend(t)
				_, err := b.HandleRequest(context.Background(), &logical.Request{
					Storage:   storage,
					Operation: logical.UpdateOperation,
					Path:      "keys/register/" + key,
					Data: map[string]interface{}{
						"crypto_key": tc.cryptoKey,
						"verify":     tc.verify,
					},
				})
				if err != nil {
					if tc.err {
						return
					}

					t.Fatal(err)
				}

				k, err := b.Key(context.Background(), storage, key)
				if err != nil {
					t.Fatal(err)
				}

				if v, exp := k.CryptoKeyID, tc.cryptoKey; v != exp {
					t.Errorf("expected %q to be %q", v, exp)
				}
			})
		}
	})
}
