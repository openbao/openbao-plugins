// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package gcpkms

import (
	"context"
	"testing"
	"time"

	"github.com/openbao/openbao/sdk/v2/logical"
)

func TestPathReencrypt_Write(t *testing.T) {

	t.Run("field_validation", func(t *testing.T) {
		testFieldValidation(t, logical.UpdateOperation, "reencrypt/my-key")
	})

	cryptoKey, cleanup := testCreateKMSCryptoKeySymmetric(t)
	defer cleanup()

	b, storage := testBackend(t)

	if err := storage.Put(context.Background(), &logical.StorageEntry{
		Key:   "keys/my-key",
		Value: []byte(`{"name":"my-key", "crypto_key_id":"` + cryptoKey + `"}`),
	}); err != nil {
		t.Fatal(err)
	}

	if err := storage.Put(context.Background(), &logical.StorageEntry{
		Key:   "keys/my-versioned-key",
		Value: []byte(`{"name":"my-versioned-key", "crypto_key_id":"` + cryptoKey + `", "min_version":3, "max_version":5}`),
	}); err != nil {
		t.Fatal(err)
	}

	t.Run("group", func(t *testing.T) {
		t.Run("integration", func(t *testing.T) {

			// Generate some ciphertext
			plaintext := "hello world"
			ctx := context.Background()
			resp, err := b.HandleRequest(ctx, &logical.Request{
				Storage:   storage,
				Operation: logical.UpdateOperation,
				Path:      "encrypt/my-key",
				Data: map[string]interface{}{
					"plaintext": plaintext,
				},
			})
			if err != nil {
				t.Fatal(err)
			}

			ciphertextV1 := resp.Data["ciphertext"].(string)

			// Rotate the key
			if _, err := b.HandleRequest(ctx, &logical.Request{
				Storage:   storage,
				Operation: logical.UpdateOperation,
				Path:      "keys/rotate/my-key",
			}); err != nil {
				t.Fatal(err)
			}

			// Wait for rotation to complete - supposedly this can take up to 3 hours, so
			// this test might be flakey and timeout.
			doneCh := make(chan struct{})
			go func() {
				base := 500 * time.Millisecond

				for {
					select {
					case <-doneCh:
						return
					case <-time.After(base):
					}

					ctx := context.Background()
					resp, err := b.HandleRequest(ctx, &logical.Request{
						Storage:   storage,
						Operation: logical.ReadOperation,
						Path:      "keys/my-key",
					})
					if err != nil {
						t.Fatal(err)
					}

					pv, ok := resp.Data["primary_version"]
					if !ok {
						t.Fatal("missing primary_version key")
					}

					pvs, ok := pv.(string)
					if !ok {
						t.Fatal("primary_version is not a string")
					}

					if pvs == "2" {
						close(doneCh)
						return
					}

					base = base * base
				}
			}()

			select {
			case <-time.After(15 * time.Second):
				t.Fatal("test timed out")
				close(doneCh)
			case <-doneCh:
				// Rotation finished
			}

			// Encrypt the ciphertext
			encryptResp, err := b.HandleRequest(ctx, &logical.Request{
				Storage:   storage,
				Operation: logical.UpdateOperation,
				Path:      "reencrypt/my-key",
				Data: map[string]interface{}{
					"ciphertext": ciphertextV1,
				},
			})
			if err != nil {
				t.Fatal(err)
			}

			kv, ok := encryptResp.Data["key_version"]
			if !ok {
				t.Fatal("missing key_version")
			}
			kvs, ok := kv.(string)
			if !ok {
				t.Fatal("key_version is not a string")
			}
			if kvs != "2" {
				t.Errorf("wrong key version")
			}

			ciphertextV2 := encryptResp.Data["ciphertext"].(string)

			if ciphertextV1 == ciphertextV2 {
				t.Errorf("not reencrypted")
			}
		})

		t.Run("less_min_version", func(t *testing.T) {

			ctx := context.Background()
			_, err := b.HandleRequest(ctx, &logical.Request{
				Storage:   storage,
				Operation: logical.UpdateOperation,
				Path:      "reencrypt/my-versioned-key",
				Data: map[string]interface{}{
					"ciphertext":  "hello world",
					"key_version": 2,
				},
			})
			if err != logical.ErrPermissionDenied {
				t.Errorf("expected %q to be %q", err, logical.ErrPermissionDenied)
			}
		})

		t.Run("greater_max_version", func(t *testing.T) {

			ctx := context.Background()
			_, err := b.HandleRequest(ctx, &logical.Request{
				Storage:   storage,
				Operation: logical.UpdateOperation,
				Path:      "reencrypt/my-versioned-key",
				Data: map[string]interface{}{
					"ciphertext":  "hello world",
					"key_version": 7,
				},
			})
			if err != logical.ErrPermissionDenied {
				t.Errorf("expected %q to be %q", err, logical.ErrPermissionDenied)
			}
		})
	})
}
