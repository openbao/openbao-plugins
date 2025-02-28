// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package gcpkms

import (
	"context"
	"testing"

	"github.com/openbao/openbao/sdk/v2/logical"
)

func TestPathKeysRotate_Write(t *testing.T) {

	t.Run("field_validation", func(t *testing.T) {
		testFieldValidation(t, logical.UpdateOperation, "keys/rotate/my-key")
	})

	cryptoKey, cleanup := testCreateKMSCryptoKeySymmetric(t)
	defer cleanup()

	b, storage := testBackend(t)

	if err := storage.Put(context.Background(), &logical.StorageEntry{
		Key:   "keys/key-without-crypto-key",
		Value: []byte(`{"name":"my-key", "crypto_key_id":"not-a-real-cryptokey"}`),
	}); err != nil {
		t.Fatal(err)
	}
	if err := storage.Put(context.Background(), &logical.StorageEntry{
		Key:   "keys/my-key",
		Value: []byte(`{"name":"my-key", "crypto_key_id":"` + cryptoKey + `"}`),
	}); err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		name string
		key  string
		err  bool
	}{
		{
			"key_not_exist",
			"not-a-real-key",
			true,
		},
		{
			"crypto_key_not_exist",
			"key-without-crypto-key",
			true,
		},
		{
			"success",
			"my-key",
			false,
		},
	}

	t.Run("group", func(t *testing.T) {
		for _, tc := range cases {
			tc := tc

			t.Run(tc.name, func(t *testing.T) {

				ctx := context.Background()
				resp, err := b.HandleRequest(ctx, &logical.Request{
					Storage:   storage,
					Operation: logical.UpdateOperation,
					Path:      "keys/rotate/" + tc.key,
				})
				if err != nil {
					if tc.err {
						return
					}
					t.Fatal(err)
				}

				if v, exp := resp.Data["key_version"].(string), "2"; v != exp {
					t.Errorf("expected %q to be %q", v, exp)
				}
			})
		}
	})
}
