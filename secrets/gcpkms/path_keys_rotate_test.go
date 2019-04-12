package gcpkms

import (
	"context"
	"testing"

	"github.com/hashicorp/vault/sdk/logical"
)

func TestPathKeysRotate_Write(t *testing.T) {
	t.Parallel()

	t.Run("field_validation", func(t *testing.T) {
		t.Parallel()
		testFieldValidation(t, logical.CreateOperation, "keys/rotate/my-key")
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
				t.Parallel()

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
